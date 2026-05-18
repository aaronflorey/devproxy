package main

import (
	"os"
	"runtime"

	"github.com/mochaka/devproxy/cmd/devproxy"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "menubar" {
		runtime.LockOSThread()
	}
	if err := devproxy.Execute(); err != nil {
		os.Exit(1)
	}
}
