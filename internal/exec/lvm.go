package exec

import (
	"fmt"
)

// CreateLV creates a logical volume with the given name, volume group, and size in megabytes.
// Runs: lvcreate -n <name> -L <size>M <vg>
func CreateLV(vg, name string, sizeMB int) error {
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
