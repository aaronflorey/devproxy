package main

import (
	"os"

	"github.com/mochaka/devproxy/cmd/devproxy"
)

func main() {
	if err := devproxy.Execute(); err != nil {
		os.Exit(1)
	}
}
