//go:build !unix

package main

import "os"

// termSize 在非 unix 平台上拿不到终端尺寸，一律返回 0：
// 进度条退回"不占行、单行原地刷新"的模式，功能不受影响。
func termSize(*os.File) (rows, cols int) { return 0, 0 }
