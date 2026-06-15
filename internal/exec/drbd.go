package exec

import (
	"fmt"
	"os"
	"path/filepath"
)

const drbdConfDir = "/etc/drbd.d"

// WriteRes writes a DRBD resource file to /etc/drbd.d/<name>.res.
func WriteRes(name, content string) error {
	path := filepath.Join(drbdConfDir, name+".res")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write resource file %q: %w", path, err)
	}
	return nil
}

// RemoveRes removes a DRBD resource file from /etc/drbd.d/<name>.res.
func RemoveRes(name string) error {
	path := filepath.Join(drbdConfDir, name+".res")
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove resource file %q: %w", path, err)
	}
	return nil
}

// Up brings a DRBD resource up.
// Runs: drbdadm up <resource>
func Up(resource string) error {
	return runCmd("drbdadm", "up", resource)
}

// Down brings a DRBD resource down.
// Runs: drbdadm down <resource>
func Down(resource string) error {
	return runCmd("drbdadm", "down", resource)
}

// Primary promotes a DRBD resource to primary role.
// Runs: drbdadm primary <resource>
func Primary(resource string) error {
	return runCmd("drbdadm", "primary", resource)
}

// Secondary demotes a DRBD resource to secondary role.
// Runs: drbdadm secondary <resource>
func Secondary(resource string) error {
	return runCmd("drbdadm", "secondary", resource)
}

// Resize resizes a DRBD resource.
// Runs: drbdadm resize <resource>
func Resize(resource string) error {
	return runCmd("drbdadm", "resize", resource)
}

// Status returns the status output for a DRBD resource.
// Runs: drbdadm status <resource>
func Status(resource string) (string, error) {
	return runCmdOutput("drbdadm", "status", resource)
}
