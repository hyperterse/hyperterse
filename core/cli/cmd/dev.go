package cmd

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/spf13/cobra"
)

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Run the Hyperterse server in development mode",
	Long:  `Run the Hyperterse server and restart it when the config file changes.`,
	RunE:  runDevServer,
}

func init() {
	rootCmd.AddCommand(devCmd)
	devCmd.Flags().StringVarP(&port, "port", "p", "", "Server port (overrides config file and PORT env var)")
	devCmd.Flags().IntVar(&logLevel, "log-level", 0, "Log level: 1=ERROR, 2=WARN, 3=INFO, 4=DEBUG (overrides config file)")
	devCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging (sets log level to DEBUG)")
}

func runDevServer(cmd *cobra.Command, args []string) error {
	log := logger.New("dev")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	if err := watcher.Add(configFile); err != nil {
		return err
	}

	restart := make(chan struct{}, 1)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Watch for file changes
	go func() {
		var debounce *time.Timer
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					if debounce != nil {
						debounce.Stop()
					}
					debounce = time.AfterFunc(500*time.Millisecond, func() {
						select {
						case restart <- struct{}{}:
						default:
						}
					})
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	log.Printf("Watching %s for changes...", configFile)

	for {
		rt, err := PrepareRuntime()
		if err != nil {
			return err
		}

		if err := rt.StartAsync(); err != nil {
			return err
		}

		select {
		case <-sigChan:
			return rt.Stop()
		case <-restart:
			log.Println("Config changed, restarting...")
			rt.Stop()
		}
	}
}
