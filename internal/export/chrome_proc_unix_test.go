//go:build unix

package export

import (
	"os/exec"
	"strconv"
	"strings"
)

func processesHolding(profile string) []int {
	out, err := exec.Command("ps", "-eo", "pid,args").Output()
	if err != nil || profile == "" {
		return nil
	}
	var pids []int
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, profile) {
			continue
		}
		pidText, _, _ := strings.Cut(line, " ")
		n, err := strconv.Atoi(pidText)
		if err == nil && n > 0 {
			pids = append(pids, n)
		}
	}
	return pids
}
