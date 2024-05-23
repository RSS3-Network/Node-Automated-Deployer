package main

import (
	"github.com/rss3-network/node-compose/pkg/cmd"
	"os"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
