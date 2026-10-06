package main

import (
	"os"

	"github.com/mahirisikli/memento/internal/cli"
)

var version = "dev"

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
