package exec

import (
	"fmt"
)

// CreateLV creates a logical volume with the given name, volume group, and size in megabytes.
// When thinpool is non-empty, creates a thin LV from that pool instead of a regular LV.
func CreateLV(vg, name, thinpool string, sizeMB int) error {
	if thinpool != "" {
		return runCmd("lvcreate", "-n", name, "-V", fmt.Sprintf("%dM", sizeMB), "--thinpool", vg+"/"+thinpool)
	}
	return runCmd("lvcreate", "-n", name, "-L", fmt.Sprintf("%dM", sizeMB), vg)
}

// ExtendLV extends a logical volume by the given number of megabytes.
// Runs: lvextend -L +<add>M /dev/<vg>/<name>
func ExtendLV(vg, name string, addMB int) error {
	return runCmd("lvextend", "-L", fmt.Sprintf("+%dM", addMB), fmt.Sprintf("/dev/%s/%s", vg, name))
}

// RemoveLV removes a logical volume.
// Runs: lvremove -f /dev/<vg>/<name>
func RemoveLV(vg, name string) error {
	return runCmd("lvremove", "-f", fmt.Sprintf("/dev/%s/%s", vg, name))
}
