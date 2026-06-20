package exec

import (
	"fmt"
	"os/exec"
	"strings"
)

// NBDServe starts a qemu-nbd server for the given device on the given TCP port.
// The process daemonizes via --fork, so this returns once the server is ready.
// Any existing qemu-nbd on the same port is killed first.
func NBDServe(device string, port int) error {
	// Kill any stale server on this port before starting a new one.
	NBDStop(port) //nolint:errcheck

	return runCmd("qemu-nbd",
		"--format", "raw",
		"--fork",
		"--persistent",
		"--bind", "0.0.0.0",
		"--port", fmt.Sprintf("%d", port),
		device,
	)
}

// NBDStop kills the qemu-nbd process serving on the given port.
func NBDStop(port int) error {
	portStr := fmt.Sprintf("%d", port)
	out, err := exec.Command("pgrep", "-f", "qemu-nbd.*--port.*"+portStr).Output()
	if err != nil {
		// No matching process — already stopped.
		return nil
	}
	pids := strings.Fields(strings.TrimSpace(string(out)))
	for _, pid := range pids {
		exec.Command("kill", pid).Run() //nolint:errcheck
	}
	return nil
}
