//go:build windows

package export

import (
	"os"
	"os/exec"
	"syscall"
	"unsafe"
)

func prepareChromeProcess(cmd *exec.Cmd) {}

// finishChromeTree 收掉浏览器进程树。Windows 的 TerminateProcess 只作用于
// 一个 PID，主进程没了之后渲染子进程还活着，必须按父 PID 再扫一遍。
func finishChromeTree(pid int, includeRoot bool) {
	if pid <= 0 {
		return
	}
	if includeRoot {
		terminatePID(pid)
	}
	for pass := 0; pass < 3; pass++ {
		kids := descendantPIDs(pid)
		if len(kids) == 0 {
			return
		}
		for _, child := range kids {
			terminatePID(child)
		}
	}
}

func terminatePID(pid int) {
	if pid <= 0 || pid == os.Getpid() {
		return
	}
	h, err := syscall.OpenProcess(syscall.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return
	}
	_ = syscall.TerminateProcess(h, 1)
	syscall.CloseHandle(h)
}

func chromeStillRunning(pid int) bool {
	return processAlive(pid)
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	h, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(h)
	var code uint32
	if err := syscall.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	return code == 259 // STILL_ACTIVE
}

func treeAlive(pid int) bool {
	if processAlive(pid) {
		return true
	}
	return len(descendantPIDs(pid)) > 0
}

func descendantPIDs(root int) []int {
	if root <= 0 {
		return nil
	}
	snap, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil
	}
	defer syscall.CloseHandle(snap)
	var e syscall.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	if err := syscall.Process32First(snap, &e); err != nil {
		return nil
	}
	children := make(map[int][]int)
	for {
		pid := int(e.ProcessID)
		ppid := int(e.ParentProcessID)
		if pid > 0 && ppid > 0 {
			children[ppid] = append(children[ppid], pid)
		}
		if err := syscall.Process32Next(snap, &e); err != nil {
			break
		}
	}
	var out []int
	seen := map[int]bool{root: true}
	stack := append([]int(nil), children[root]...)
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if n <= 0 || seen[n] || n == os.Getpid() {
			continue
		}
		seen[n] = true
		out = append(out, n)
		stack = append(stack, children[n]...)
	}
	return out
}
