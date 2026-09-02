package unit

// Look 是一种 kind 怎么画。角色在 Register 时填好，快照里原样发给页面。
// 前端按这些字段画，不要再按 Kind 名字分支。
type Look struct {
	Color      string  `json:"color,omitempty"`
	Chroma     bool    `json:"chroma,omitempty"`
	Ghost      float64 `json:"ghost,omitempty"`
	Trail      bool    `json:"trail,omitempty"`
	Glow       bool    `json:"glow,omitempty"`
	VisionRing bool    `json:"visionRing,omitempty"`
	Field      string  `json:"field,omitempty"`
	Bond       bool    `json:"bond,omitempty"`
	BondColor  string  `json:"bondColor,omitempty"`
	WallGuide  float64 `json:"wallGuide,omitempty"`
	Ring       bool    `json:"ring,omitempty"`
}

func Looks() map[string]Look {
	out := make(map[string]Look, len(specs))
	for k, s := range specs {
		out[k] = s.Look
	}
	return out
}
