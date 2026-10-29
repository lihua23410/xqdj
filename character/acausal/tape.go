package 非因果

import "xqdj/internal/unit"

// 探针协议放在 unit，避免出生参数依赖具体战斗机包。
type Sample = unit.Sample
type FighterTrack = unit.FighterTrack
type MinionTrack = unit.MinionTrack
type Tape = unit.ProbeTape
