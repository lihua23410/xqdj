package unit

// Mark 是打在单位快照上的可叠加状态，角色自己定 Kind / Icon，引擎只负责层数和每层寿命。
type Mark struct {
	Kind   string `json:"kind"`
	Stacks int    `json:"stacks"`
	Icon   string `json:"icon,omitempty"`
	// Lifetime > 0 时，每一层从叠上去那一拍起活这么多秒；0 表示不过期。
	Lifetime float64 `json:"lifetime,omitempty"`
	// NextDecayAt 是最老那一层的过期时间；Lifetime = 0 时为 0。
	NextDecayAt float64 `json:"nextDecayAt,omitempty"`
	// Hidden 的标记只给角色自己的演出看，不进通用状态 HUD。
	Hidden bool `json:"hidden,omitempty"`
}

type StackMark struct {
	UnitID uint64
	Kind   string
	Delta  int
	Icon   string
	// Lifetime > 0 时，每一层从叠上去那一拍起活这么多秒；0 表示不过期。
	Lifetime float64
	// Hidden 的标记只给角色自己的演出看，不进通用状态 HUD。
	Hidden bool
}

type ClearMarks struct {
	UnitID uint64
	Kind   string // 空则清掉该单位全部标记
}
