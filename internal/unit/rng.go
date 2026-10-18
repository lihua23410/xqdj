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

// NextStream 向根 PCG 要一个数。只有每局开赛的比赛种子走这里。
func NextStream() uint64 {
	rootMu.Lock()
	defer rootMu.Unlock()
	return root.Uint64()
}

const (
	StreamRoot    = 1
	StreamChild   = 2
	StreamFaction = 3
)

func NewRNG(stream uint64) *rand.Rand {
	return rand.New(rand.NewPCG(stream, stream^0x9e3779b97f4a7c15))
}

// DeriveStream 由父流号和第几个孩子算出子流号。只看这两个数，不看谁先被引擎处理。
func DeriveStream(parent, n, domain uint64) uint64 {
	r := rand.New(rand.NewPCG(parent^domain, n))
	return r.Uint64()
}

// EngineRNG 是比赛引擎那只。流号跟这局种子绑死，落点和换位能用种子对上。
func EngineRNG(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
}
