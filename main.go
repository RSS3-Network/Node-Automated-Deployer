package main

import (
	"os"

	"github.com/rss3-network/node-compose/pkg/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
