package unit

import "math/rand/v2"

// NewRNG 给角色一条独立流。运行时熵拌 ID，避免只靠槽位或出场顺序把随机钉死。
// 测试先塞好 *rand.Rand 就不要再调这个。
func NewRNG(id uint64) *rand.Rand {
	return rand.New(rand.NewPCG(rand.Uint64()^id, rand.Uint64()))
}
