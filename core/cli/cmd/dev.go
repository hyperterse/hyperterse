package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/spf13/cobra"
)

var devCmd = &cobra.Command{
	Use:          "dev",
	Short:        "Run the Hyperterse server in development mode",
	Long:         `Run the Hyperterse server and restart it when the config file changes.`,
	RunE:         runDevServer,
	SilenceUsage: true,
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

	log.Infof("Watching %s for changes", configFile)

	// Start initial runtime - fail immediately if this doesn't work
	rt, err := PrepareRuntime()
	if err != nil {
		return err
	}

	if err := rt.StartAsync(); err != nil {
		return err
	}

	for {
		select {
		case <-sigChan:
			return rt.Stop()
		case <-restart:
			log.Infof("Config changed, reloading")

			// Try to prepare new runtime first (before stopping old one)
			// This allows the old server to keep running if config is invalid
			newRt, err := PrepareRuntime()
			if err != nil {
				log.Errorf("Failed to load new config, keeping current server running: %v", err)
				continue
			}

			// Stop old runtime before starting new one
			if err := rt.Stop(); err != nil {
				log.Errorf("Failed to stop server gracefully: %v", err)
				// Return error since we can't safely start new server if old one didn't stop
				// (would cause "address already in use" error)
				return fmt.Errorf("failed to stop server for reload: %w", err)
			}

			// Start new runtime
			if err := newRt.StartAsync(); err != nil {
				log.Errorf("Failed to start new server: %v", err)
				return err
			}

			rt = newRt
			log.Infof("Server reloaded successfully")
		}
	}
}
