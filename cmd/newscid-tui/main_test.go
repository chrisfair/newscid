package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMainVersionFlag(t *testing.T) {
	origArgs := os.Args
	origExit := exitFn
	defer func() {
		os.Args = origArgs
		exitFn = origExit
	}()

	gotCode := -1
	exitFn = func(code int) {
		gotCode = code
	}
	os.Args = []string{"newscid-tui", "--version"}
	main()
	assert.Equal(t, 0, gotCode)
}
