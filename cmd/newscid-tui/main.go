package main

import (
	"os"

	"newscid/pkg/tui"
)

var exitFn = os.Exit

func main() {
	exitFn(tui.RunCLI(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
