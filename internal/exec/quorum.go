package exec

import "strings"

// ClusterQuorate returns true if the local Pacemaker/Corosync cluster has quorum.
// Runs corosync-quorumtool and checks the "Quorate:" field.
func ClusterQuorate() (bool, error) {
	out, err := runCmdOutput("corosync-quorumtool", "-s")
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "Quorate:") {
			return strings.Contains(line, "Yes"), nil
		}
	}
	return false, nil
}
