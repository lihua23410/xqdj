package unit

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"math/rand/v2"
	"sync"
)

// 进程里一只根 PCG。启动时用系统熵种一次，之后每颗比赛种子、每条单位流都从它吐出来。
var (
	rootMu sync.Mutex
	root   = seedRoot()
)

func seedRoot() *rand.Rand {
	var buf [16]byte
	if _, err := cryptorand.Read(buf[:]); err != nil {
		panic(err)
	}
	s0 := binary.LittleEndian.Uint64(buf[:8])
	s1 := binary.LittleEndian.Uint64(buf[8:])
	return rand.New(rand.NewPCG(s0, s1))
}

// NextStream 向根 PCG 要一个数。比赛开局的种子、单位出生时的流号都走这里。
func NextStream() uint64 {
	rootMu.Lock()
	defer rootMu.Unlock()
	return root.Uint64()
}

// NewRNG 给角色一条独立流。流号当场从根 PCG 取，不看单位 ID：ID 每局从 1 重数，用它会把同一对阵连开打成同一局。
// id 留着是为了现有调用点不用改。测试先塞好 *rand.Rand 就不要再调这个。
func NewRNG(id uint64) *rand.Rand {
	_ = id
	s := NextStream()
	return rand.New(rand.NewPCG(s, s^0x9e3779b97f4a7c15))
}

// EngineRNG 是比赛引擎那只。流号跟这局种子绑死，落点和换位能用种子对上。
func EngineRNG(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
}
