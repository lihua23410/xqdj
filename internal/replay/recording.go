// Package replay 定义观众录像，独立于模拟引擎和视频编码。
package replay

import (
	"encoding/json"
	"math"
)

type IntroBody struct {
	K string  `json:"k"`
	S int     `json:"s"`
	X float64 `json:"x"`
	Y float64 `json:"y"`
	R float64 `json:"r"`
}

type IntroFrame struct {
	T float64     `json:"t"`
	B []IntroBody `json:"b"`
}

type IntroFilm struct {
	Death  float64      `json:"death"`
	Frames []IntroFrame `json:"frames"`
}

// Duration 不含服务端等待浏览器回执的超时余量。
func (f *IntroFilm) Duration() float64 {
	if f == nil || len(f.Frames) == 0 || f.Death < 0 || math.IsNaN(f.Death) || math.IsInf(f.Death, 0) {
		return 0
	}
	return 1 + f.Death/10
}

// Recording 的 JSON 和演出数据在发布后只读。Frames 不重复存放演出轨迹。
type Recording struct {
	Frames  []json.RawMessage
	Initial json.RawMessage
	Intro   *IntroFilm
}
