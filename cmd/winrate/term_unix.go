//go:build unix

package main

import (
	"os"
	"syscall"
	"unsafe"
)

// termSize 读终端的行列数。拿不到就返回 0，调用方退回单行刷新。
func termSize(f *os.File) (rows, cols int) {
	ws := struct{ row, col, xpixel, ypixel uint16 }{}
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(),
		uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&ws))); errno != 0 || ws.row == 0 {
		return 0, 0
	}
	return int(ws.row), int(ws.col)
}
