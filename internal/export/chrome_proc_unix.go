//go:build unix

package export

import (
	"os/exec"
	"syscall"
)

func prepareChromeProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// finishChromeTree 收掉浏览器进程组。启动时设了 Setpgid，子进程和主进程同组，
// 主进程先死后组还在，向组发 SIGKILL 才能带走渲染进程。
func finishChromeTree(pid int, includeRoot bool) {
	if pid <= 0 {
		return
	}
	_ = syscall.Kill(-pid, syscall.SIGKILL)
	if includeRoot {
		_ = syscall.Kill(pid, syscall.SIGKILL)
	}
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}

// chromeStillRunning 把僵尸进程当成已经退出。Wait4 用 WNOHANG，跑着的进程不会被收走。
func chromeStillRunning(pid int) bool {
	if !processAlive(pid) {
		return false
	}
	var status syscall.WaitStatus
	wpid, err := syscall.Wait4(pid, &status, syscall.WNOHANG, nil)
	return !(err == nil && wpid == pid)
}

func treeAlive(pid int) bool {
	if processAlive(pid) {
		return true
	}
	return syscall.Kill(-pid, 0) == nil
}
