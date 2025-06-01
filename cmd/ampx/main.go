package main

import (
	"os"

	"github.com/brettsmith212/ci-test-2/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
