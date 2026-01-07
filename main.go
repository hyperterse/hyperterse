package main

import (
	"os"

	"github.com/hyperterse/hyperterse/core/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
