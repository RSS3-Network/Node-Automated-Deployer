package main

import (
	"os"

	"github.com/rss3-network/node-automated-deployer/pkg/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
