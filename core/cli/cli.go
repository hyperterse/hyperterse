package cli

import (
	"github.com/hyperterse/hyperterse/core/cli/cmd"
	"github.com/hyperterse/hyperterse/core/logger"
)

// Execute runs the CLI
func Execute() error {
	if err := cmd.Execute(); err != nil {
		tag := logger.ErrorTag(err)
		if tag == "" {
			tag = "cli"
		}
		logger.New(tag).Error(err.Error())
		return err
	}
	return nil
}
