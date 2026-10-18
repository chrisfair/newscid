package main

import (
	"os"

	"newscid/pkg/workstation"
)

var exitFn = os.Exit

func main() {
	exitFn(workstation.RunImportCLI(os.Args[1:], os.Stdout, os.Stderr))
}
