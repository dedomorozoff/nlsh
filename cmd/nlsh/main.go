package main

import (
	"os"
	"runtime"

	"github.com/dedomorozoff/nlsh/internal/cli"
)

func init() {
	if runtime.GOOS == "windows" {
		os.Setenv("CHCP", "65001")
	}
}

func main() {
	cmd := cli.NewRootCmd()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
