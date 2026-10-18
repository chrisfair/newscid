package main

import (
	"os"
	"testing"
)

func TestMainVersion(t *testing.T) {
	origArgs := os.Args
	origExit := exitFn
	defer func() {
		os.Args = origArgs
		exitFn = origExit
	}()
	var gotCode int
	exitFn = func(code int) { gotCode = code }
	os.Args = []string{"newscid-import", "-version"}
	main()
	if gotCode != 0 {
		t.Fatalf("expected exit code 0, got %d", gotCode)
	}
}
