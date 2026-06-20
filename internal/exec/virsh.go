package exec

import (
	"fmt"
	"strings"
)

// VMBlockList returns the virtio block device targets currently attached to a VM.
// Parses: virsh domblklist <domain>
func VMBlockList(domain string) ([]string, error) {
	out, err := runCmdOutput("virsh", "domblklist", domain)
	if err != nil {
		return nil, fmt.Errorf("virsh domblklist %s: %w", domain, err)
	}

	var targets []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Target") || strings.HasPrefix(line, "-") {
			continue
		}
		if fields := strings.Fields(line); len(fields) >= 1 {
			targets = append(targets, fields[0])
		}
	}
	return targets, nil
}

// VMNextTarget returns the next free virtio block target ("vdb", "vdc", …)
// given a set of already-used target names.
func VMNextTarget(used []string) string {
	inUse := make(map[string]bool, len(used))
	for _, t := range used {
		inUse[t] = true
	}
	for _, c := range "bcdefghijklmnopqrstuvwxyz" {
		if t := "vd" + string(c); !inUse[t] {
			return t
		}
	}
	return ""
}

// VMAttach hot-attaches a block device to a running libvirt domain.
// Calls: virsh attach-disk <domain> <source> <target> --live
func VMAttach(domain, source, target string) error {
	return runCmd("virsh", "attach-disk", domain, source, target, "--live")
}

// VMDetach hot-detaches a virtio disk from a running libvirt domain by target name.
// Calls: virsh detach-disk <domain> <target> --live
func VMDetach(domain, target string) error {
	return runCmd("virsh", "detach-disk", domain, target, "--live")
}
