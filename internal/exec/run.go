package exec

import (
	"bytes"
	"fmt"
	osexec "os/exec"
	"strings"
)

// runCmd runs a command and returns an error that includes stderr output on failure.
func runCmd(name string, args ...string) error {
	var stderr bytes.Buffer
	cmd := osexec.Command(name, args...)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return fmt.Errorf("command %q failed: %w", name, err)
		}
		return fmt.Errorf("command %q failed: %s", name, msg)
	}
	return nil
}

// runCmdOutput runs a command and returns stdout, or an error that includes stderr on failure.
func runCmdOutput(name string, args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := osexec.Command(name, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return "", fmt.Errorf("command %q failed: %w", name, err)
		}
		return "", fmt.Errorf("command %q failed: %s", name, msg)
	}
	return stdout.String(), nil
}
