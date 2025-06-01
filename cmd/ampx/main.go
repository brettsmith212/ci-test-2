package main

import (
	"os"

	"github.com/brettsmith212/ci-test-2/internal/cli"
	"github.com/brettsmith212/ci-test-2/internal/cli/commands"
)

func main() {
	// Register all commands
	cli.AddCommand(commands.NewStartCommand())
	cli.AddCommand(commands.NewListCommand())
	cli.AddCommand(commands.NewLogsCommand())
	cli.AddCommand(commands.NewContinueCommand())
	cli.AddCommand(commands.NewAbortCommand())
	cli.AddCommand(commands.NewMergeCommand())

	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
