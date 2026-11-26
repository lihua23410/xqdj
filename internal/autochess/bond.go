package autochess

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"xqdj/internal/sim"
)

// 一组两到四人。人数按场上不同的战斗机计，同一只再多也不加，备战席不算。
// 够到的最高一档盖过较低档，不叠乘。效果只给本组这一回在场的战斗机。
type bondDef struct {
	Name    string
	Members []string
	Tiers   []bondTier
}

type bondTier struct {
	Need   int
	HP     float64
	Dmg    float64
	Speed  float64
	Shield float64
}

var bondBook = []bondDef{
	{Name: "老头组", Members: []string{"炭翁", "酒翁", "谭泳辰老先生"}, Tiers: []bondTier{
		{Need: 2, HP: 1.12, Shield: 0.08},
		{Need: 3, HP: 1.22, Shield: 0.15},
	}},
	{Name: "月计组", Members: []string{"地慧星", "R.缪", "钉与锤"}, Tiers: []bondTier{
		{Need: 2, Dmg: 1.12, Speed: 1.08},
		{Need: 3, Dmg: 1.22, Speed: 1.12},
	}},
	{Name: "原型机组", Members: []string{"原型机_远程", "原型机_近战", "原型机_整合"}, Tiers: []bondTier{
		{Need: 2, Speed: 1.1},
		{Need: 3, Speed: 1.16, Dmg: 1.1},
	}},
	{Name: "百战组", Members: []string{"分身者(百战)", "雷达(百战)", "筑墙者(百战)"}, Tiers: []bondTier{
		{Need: 2, Dmg: 1.12},
		{Need: 3, Dmg: 1.22, HP: 1.1},
	}},
	{Name: "怪猎组", Members: []string{"弓箭", "太刀", "盾斧"}, Tiers: []bondTier{
		{Need: 2, Dmg: 1.18},
		{Need: 3, Dmg: 1.32},
	}},
	{Name: "西幻组", Members: []string{"血契", "收割者", "狼人"}, Tiers: []bondTier{
		{Need: 2, HP: 1.1, Dmg: 1.1},
		{Need: 3, HP: 1.18, Dmg: 1.18},
	}},
	{Name: "主角组", Members: []string{"勇者", "小骑士", "教父(青年)"}, Tiers: []bondTier{
		{Need: 2, Dmg: 1.12, Speed: 1.08},
		{Need: 3, Dmg: 1.2, Speed: 1.12},
	}},
	{Name: "神秘组", Members: []string{"迷走", "我与周旋久", "时计", "时计·铭刻"}, Tiers: []bondTier{
		{Need: 2, HP: 1.12},
		{Need: 3, HP: 1.24, Shield: 0.1},
	}},
	{Name: "耍起组", Members: []string{"钓鱼佬", "蛆", "人偶使"}, Tiers: []bondTier{
		{Need: 2, Dmg: 1.15},
		{Need: 3, Dmg: 1.28},
	}},
	{Name: "犯罪组", Members: []string{"囚徒", "教父", "吹笛人"}, Tiers: []bondTier{
		{Need: 2, Shield: 0.12},
		{Need: 3, Shield: 0.2, HP: 1.1},
	}},
	{Name: "工程组", Members: []string{"筑墙者", "内燃机", "中转站"}, Tiers: []bondTier{
		{Need: 2, HP: 1.15},
		{Need: 3, HP: 1.28},
	}},
	{Name: "安保组", Members: []string{"盾士", "雷达", "分身者"}, Tiers: []bondTier{
		{Need: 2, HP: 1.18},
		{Need: 3, HP: 1.32, Shield: 0.12},
	}},
	{Name: "艺教组", Members: []string{"老牧师", "面灵气", "无下限术士"}, Tiers: []bondTier{
		{Need: 2, Dmg: 1.12},
		{Need: 3, Dmg: 1.24},
	}},
	{Name: "肥鱼组", Members: []string{"大肥鱼", "大肥鱼(幻方量化)", "泡泡球"}, Tiers: []bondTier{
		{Need: 2, HP: 1.15},
		{Need: 3, HP: 1.3},
	}},
	{Name: "专线组", Members: []string{"中转站[ospf]", "中转站[Freischutz]", "观察者"}, Tiers: []bondTier{
		{Need: 2, Dmg: 1.12},
		{Need: 3, Dmg: 1.22},
	}},
	{Name: "机枢组", Members: []string{"内燃机(泵之心)", "阈值", "静电魔球", "凝聚体"}, Tiers: []bondTier{
		{Need: 2, Dmg: 1.12},
		{Need: 3, Dmg: 1.24},
	}},
	{Name: "幻戏组", Members: []string{"藤姬", "小幺哥", "反重力猫咪"}, Tiers: []bondTier{
		{Need: 2, Speed: 1.1},
		{Need: 3, Speed: 1.16, Dmg: 1.1},
	}},
	{Name: "残影组", Members: []string{"回声", "闪烁", "非因果"}, Tiers: []bondTier{
		{Need: 2, Speed: 1.12},
		{Need: 3, Speed: 1.2},
	}},
	{Name: "案牍组", Members: []string{"探长", "传票员", "外科手术"}, Tiers: []bondTier{
		{Need: 2, Dmg: 1.1, Shield: 0.08},
		{Need: 3, Dmg: 1.2, Shield: 0.12},
	}},
	{Name: "落点组", Members: []string{"纵火者", "巡回", "捆"}, Tiers: []bondTier{
		{Need: 2, Dmg: 1.12},
		{Need: 3, Dmg: 1.22},
	}},
	{Name: "彩头组", Members: []string{"老虎机", "粉红色迷人的pink"}, Tiers: []bondTier{
		{Need: 2, Dmg: 1.2},
	}},
}

var (
	bondByKind = map[string]string{}
	bondByName = map[string]bondDef{}
)

func init() {
	for _, b := range bondBook {
		bondByName[b.Name] = b
		for _, kind := range b.Members {
			if prev, ok := bondByKind[kind]; ok {
				panic("bond: " + kind + " in both " + prev + " and " + b.Name)
			}
			bondByKind[kind] = b.Name
		}
	}
}

func bondName(kind string) string {
	return bondByKind[kind]
}

func uniqueBondCount(board []piece, name string) int {
	seen := map[string]bool{}
	for _, p := range board {
		if bondByKind[p.kind] != name || seen[p.kind] {
			continue
		}
		seen[p.kind] = true
	}
	return len(seen)
}

func reachedTier(def bondDef, count int) (bondTier, bool) {
	var got bondTier
	ok := false
	for _, t := range def.Tiers {
		if count >= t.Need {
			got = t
			ok = true
		}
	}
	return got, ok
}

func nextTierNeed(def bondDef, count int) int {
	for _, t := range def.Tiers {
		if count < t.Need {
			return t.Need
		}
	}
	return 0
}

func modsFor(board []piece, kind string) sim.PieceMods {
	name := bondByKind[kind]
	def, ok := bondByName[name]
	if !ok {
		return sim.PieceMods{}
	}
	t, ok := reachedTier(def, uniqueBondCount(board, name))
	if !ok {
		return sim.PieceMods{}
	}
	return sim.PieceMods{HP: t.HP, Dmg: t.Dmg, Speed: t.Speed, Shield: t.Shield}
}

func bondMates(kind string, owned []piece) int {
	name := bondByKind[kind]
	if name == "" {
		return 0
	}
	seen := map[string]bool{}
	for _, o := range owned {
		if o.kind == kind || bondByKind[o.kind] != name || seen[o.kind] {
			continue
		}
		seen[o.kind] = true
	}
	return len(seen)
}

func (t bondTier) line() string {
	var parts []string
	if t.HP > 1 {
		parts = append(parts, "生命"+formatMul(t.HP))
	}
	if t.Dmg > 1 {
		parts = append(parts, "伤害"+formatMul(t.Dmg))
	}
	if t.Speed > 1 {
		parts = append(parts, "速度"+formatMul(t.Speed))
	}
	if t.Shield > 0 {
		parts = append(parts, fmt.Sprintf("护盾%d%%", int(math.Round(t.Shield*100))))
	}
	return strings.Join(parts, " ")
}

func formatMul(v float64) string {
	s := strconv.FormatFloat(v, 'f', 2, 64)
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	return "×" + s
}

type bondStateView struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
	Need  int    `json:"need,omitempty"`
	Next  int    `json:"next,omitempty"`
	Text  string `json:"text,omitempty"`
}

type bondTierView struct {
	Need int    `json:"need"`
	Text string `json:"text"`
}

type bondMemberView struct {
	Kind    string `json:"kind"`
	Cost    int    `json:"cost"`
	Owned   bool   `json:"owned,omitempty"`
	Fielded bool   `json:"fielded,omitempty"`
}

type bondInfoView struct {
	Name    string           `json:"name"`
	Tiers   []bondTierView   `json:"tiers"`
	Members []bondMemberView `json:"members"`
}

func bondStates(board []piece) []bondStateView {
	var out []bondStateView
	for _, def := range bondBook {
		n := uniqueBondCount(board, def.Name)
		if n == 0 {
			continue
		}
		st := bondStateView{Name: def.Name, Count: n, Next: nextTierNeed(def, n)}
		if t, ok := reachedTier(def, n); ok {
			st.Need = t.Need
			st.Text = t.line()
		}
		out = append(out, st)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Name < out[j].Name
	})
	if out == nil {
		return []bondStateView{}
	}
	return out
}

func bondBookView(costOf map[string]int, owned, fielded map[string]bool) []bondInfoView {
	out := make([]bondInfoView, len(bondBook))
	for i, def := range bondBook {
		info := bondInfoView{Name: def.Name, Tiers: make([]bondTierView, len(def.Tiers))}
		for j, t := range def.Tiers {
			info.Tiers[j] = bondTierView{Need: t.Need, Text: t.line()}
		}
		info.Members = make([]bondMemberView, len(def.Members))
		for j, kind := range def.Members {
			info.Members[j] = bondMemberView{
				Kind:    kind,
				Cost:    costOf[kind],
				Owned:   owned[kind],
				Fielded: fielded[kind],
			}
		}
		out[i] = info
	}
	return out
}
