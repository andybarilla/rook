package main

import (
	"os"

	"github.com/andybarilla/rook/internal/cli"
	"github.com/andybarilla/rook/internal/runner"
)

func main() {
	runner.DetectRuntime()
	if err := cli.NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
