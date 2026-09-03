package unit

// Mark 是打在单位快照上的可叠加状态，角色自己定 Kind / Icon，引擎只负责层数。
type Mark struct {
	Kind   string `json:"kind"`
	Stacks int    `json:"stacks"`
	Icon   string `json:"icon,omitempty"`
}

type StackMark struct {
	UnitID uint64
	Kind   string
	Delta  int
	Icon   string
}

type ClearMarks struct {
	UnitID uint64
	Kind   string // 空则清掉该单位全部标记
}
