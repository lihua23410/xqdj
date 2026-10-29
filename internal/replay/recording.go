// Package replay 定义观众录像，独立于模拟引擎和视频编码。
package replay

import (
	"encoding/json"
	"fmt"
	"math"
)

// Presentation 是资源包提供的开场演出。Data 仅由该包的播放器解释。
// ID 由对局在提交时赋值；Audio 为 false 时播放器的音频入口静音。
type Presentation struct {
	ID       string          `json:"id"`
	Pack     string          `json:"pack"`
	Type     string          `json:"type"`
	Duration float64         `json:"duration"`
	Audio    bool            `json:"audio"`
	Data     json.RawMessage `json:"data"`
}

func (p Presentation) Validate() error {
	if p.Pack == "" || p.Type == "" {
		return fmt.Errorf("presentation needs a pack and type")
	}
	if p.Duration <= 0 || math.IsNaN(p.Duration) || math.IsInf(p.Duration, 0) {
		return fmt.Errorf("presentation needs a finite positive duration")
	}
	if !json.Valid(p.Data) {
		return fmt.Errorf("presentation data must be valid JSON")
	}
	return nil
}

func (p Presentation) Clone() Presentation {
	p.Data = append(json.RawMessage(nil), p.Data...)
	return p
}

// Recording 的 JSON 在发布后只读。开场演出与比赛帧独立存储。
type Recording struct {
	Frames        []json.RawMessage
	Initial       json.RawMessage
	Presentations []Presentation
}
