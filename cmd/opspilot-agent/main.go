package main

import (
	"fmt"
	"os"

	"github.com/shekhar396/opspilot-agent/internal/cli"
)

func main() {
	rootCmd := cli.NewRootCommand()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
