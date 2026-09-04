package unit

// Look 是一种 kind 怎么画。角色在 Register 时填好，快照里原样发给页面。
// 前端按这些字段画，不要再按 Kind 名字分支。
// 角色私产皮肤走 FX 短名（由该包 fx/*.css|js 实现），不要往这里加字段。
type Look struct {
	Color      string   `json:"color,omitempty"`
	Ghost      float64  `json:"ghost,omitempty"`
	Trail      bool     `json:"trail,omitempty"`
	Glow       bool     `json:"glow,omitempty"`
	VisionRing bool     `json:"visionRing,omitempty"`
	WallGuide  float64  `json:"wallGuide,omitempty"`
	Ring       bool     `json:"ring,omitempty"`
	Overlay    bool     `json:"overlay,omitempty"` // 画在 #over，不被六边形裁切
	FX         []string `json:"fx,omitempty"`      // 常驻皮肤短名，如 "glitch"
	Base       string   `json:"base,omitempty"`    // "/ball/<Kind>"，随从与主人相同
}

func Looks() map[string]Look {
	out := make(map[string]Look, len(specs))
	for k, s := range specs {
		out[k] = s.Look
	}
	return out
}
