package unit

const (
	FactionCyan   = "青"
	FactionRed    = "红"
	FactionPurple = "紫"
	FactionPale   = "苍"
)

func AllFactions() []string {
	return []string{FactionCyan, FactionRed, FactionPurple, FactionPale}
}

func ValidFaction(s string) bool {
	for _, f := range AllFactions() {
		if f == s {
			return true
		}
	}
	return false
}

func PickFaction(i int) string {
	all := AllFactions()
	if len(all) == 0 {
		return ""
	}
	if i < 0 {
		i = -i
	}
	return all[i%len(all)]
}

func PickOtherFaction(current string, i int) string {
	var pool []string
	for _, f := range AllFactions() {
		if f != current {
			pool = append(pool, f)
		}
	}
	if len(pool) == 0 {
		return PickFaction(i)
	}
	if i < 0 {
		i = -i
	}
	return pool[i%len(pool)]
}

// MarkFaction 给战斗机打上派系。Faction 为空则由引擎随机。
type MarkFaction struct {
	UnitID    uint64
	Faction   string
	Cycle     bool    // 撞墙（非单位）时换成另一个派系
	AmpOut    float64 // 异派系造成伤害的倍率；0 表示不改
	AmpIn     float64 // 同派系受到伤害的倍率；0 表示不改
	Collect bool     // 记下出现过的派系，凑齐四种时按 Barrage 朝四周各生成一发
	Barrage []string // 弹种 kind 列表；速度用各 kind 的 Spec.Speed。空则凑齐后什么都不做
}

type ClearFactionSeen struct {
	UnitID uint64
}

type Heal struct {
	UnitID uint64
	Amount float64
}
