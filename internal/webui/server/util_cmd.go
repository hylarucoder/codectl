package server

import (
	"os/exec"
)

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Start()
}
