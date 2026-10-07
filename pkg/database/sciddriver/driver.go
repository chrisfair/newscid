package sciddriver

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Driver provides an interface to execute scid commands using tcscid if available on the system.
type Driver struct {
	tcscidPath string
}

func NewDriver() *Driver {
	path, err := exec.LookPath("/usr/share/scid/bin/tcscid")
	if err != nil {
		path, _ = exec.LookPath("tcscid")
	}
	return &Driver{tcscidPath: path}
}

func (d *Driver) IsAvailable() bool {
	return d.tcscidPath != ""
}

// ExecuteTcl executes a tcl script via tcscid and returns standard output.
func (d *Driver) ExecuteTcl(script string) (string, error) {
	if !d.IsAvailable() {
		return "", fmt.Errorf("tcscid binary not found on system")
	}

	cmd := exec.Command(d.tcscidPath)
	cmd.Stdin = strings.NewReader(script + "\nexit\n")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tcscid error (%v): %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// GetBaseGameCount queries tcscid for the game count of a database.
func (d *Driver) GetBaseGameCount(basePath string) (int, error) {
	script := fmt.Sprintf(`
set id [sc_base open "%s"]
puts [sc_base numGames $id]
sc_base close $id
`, basePath)

	out, err := d.ExecuteTcl(script)
	if err != nil {
		return 0, err
	}

	var count int
	_, err = fmt.Sscanf(out, "%d", &count)
	return count, err
}
