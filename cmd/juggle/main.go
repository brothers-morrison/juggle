package main

import (
	"fmt"
	"os"

	"github.com/ohare93/juggle/internal/cli"
)

// version is set at build time via -ldflags
var version = "0.2.0"

func main() {
	cli.SetVersion(version)
	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
