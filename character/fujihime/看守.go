package 藤姬

import (
	"xqdj/internal/unit"
)

// KindToseKeeper 是藤色的账房：藤姬把池子交给队友之后，层数、层数标记、每层加伤都由它维护。
//
// 它出生时不带主人（Spawn 不填 OwnerID），所以藤姬倒下它不会跟着一起被撤——
// 池子在谁身上，它就继续按池子改层数，挨一刀掉一层。这是「藤色不该是永久的」那条规则
// 唯一需要有人继续算账的地方。
//
// 它是纯记账的实体：Spec.Hidden 让它不进任何人的感知，也不上页面。
// 老鼠叼不走它、瘟疫找不到它、AOE 也够不着它——因为谁都看不见它。
const KindToseKeeper = "藤色看守"

// keeperData 是看守的出生参数：认谁的名下（owner）、跟谁（holder）。
type keeperData struct {
	owner  uint64
	holder uint64
}

type 看守 struct {
	owner  uint64
	holder uint64
	layers int
	booted bool
}

func newKeeper(v any) *看守 {
	d, _ := v.(keeperData)
	return &看守{owner: d.owner, holder: d.holder}
}

func (k *看守) Handle(ctx unit.Context, ev unit.Event) {
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	k.tick(ctx, s)
}

// tick 只做一件事：池子 → 层数。
// 持有人还挂着藤色就跟着池子改标记和加伤；池子空了、或者藤色已经搬去别人身上，就自己退场。
func (k *看守) tick(ctx unit.Context, s unit.Sense) {
	if k.holder == 0 {
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		return
	}
	snap := k.snapOf(s)
	if snap == nil || !hasTose(snap) {
		// 持有人没了，或者池子搬走了：这本账不归我了。
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		return
	}
	layers := layersOf(snap.Absorb)
	if !k.booted {
		// 第一拍只记账不发令：标记和加伤是藤姬搬池子时一起下发的，这里再发一遍会叠双。
		k.booted, k.layers = true, layers
		return
	}
	if layers == k.layers {
		return
	}
	ctx.Out <- unit.StackMark{UnitID: k.holder, Kind: toseKind, Delta: layers - k.layers, Icon: toseIcon}
	ctx.Out <- unit.SetDamageBonus{UnitID: k.holder, Amount: float64(layers) * toseDamagePerLayer}
	k.layers = layers
}

func (k *看守) snapOf(s unit.Sense) *unit.Snapshot {
	for i := range s.Nearby {
		if s.Nearby[i].ID == k.holder {
			return &s.Nearby[i]
		}
	}
	return nil
}

// hasTose 身上还挂着藤色吗。池子搬走时藤姬会清标记，看守据此退场。
func hasTose(o *unit.Snapshot) bool {
	for _, m := range o.Marks {
		if m.Kind == toseKind && m.Stacks > 0 {
			return true
		}
	}
	return false
}
