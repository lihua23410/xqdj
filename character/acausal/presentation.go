package 非因果

import (
	"encoding/json"
	"fmt"
	"math"
	"xqdj/internal/unit"
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

// RewindPresentation 准备本包的开场倒带，调度器不解释其数据和播放规则。
func RewindPresentation(f *IntroFilm) (unit.Presentation, error) {
	if f == nil || len(f.Frames) == 0 || f.Death < 0 || math.IsNaN(f.Death) || math.IsInf(f.Death, 0) {
		return unit.Presentation{}, fmt.Errorf("acausal: invalid rewind film")
	}
	data, err := json.Marshal(f)
	if err != nil {
		return unit.Presentation{}, err
	}
	return unit.Presentation{Pack: KindAcausal, Type: "rewind", Duration: 1 + f.Death/10, Data: data}, nil
}
