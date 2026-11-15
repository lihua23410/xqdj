package sim

import (
	"fmt"
	"sync"
	"testing"

	"xqdj/character"
	"xqdj/character/hff"
)

// 幻方量化盯盘会把场上的中转站整份抄进沙盘。抄出来的报文和原对局若还握着同一张
// 链路状态表，沙盘里的下一跳就会和原对局的学习并发读写，进程直接死掉。
func TestPaperOSPFDatabaseIsPrivate(t *testing.T) {
	const games = 80
	errc := make(chan error, games)
	var wg sync.WaitGroup
	sem := make(chan struct{}, 16)
	for seed := uint64(1); seed <= games; seed++ {
		wg.Add(1)
		go func(seed uint64) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					errc <- fmt.Errorf("seed %d: %v", seed, r)
				}
			}()
			left, right := hff.KindHFF, character.KindOSPF
			if seed%2 == 0 {
				left, right = right, left
			}
			m := NewHeadlessMatchSeeded(seed)
			m.skipProbe = true
			m.SetSlot(0, left)
			m.SetSlot(1, right)
			m.Start()
			m.Play(36000)
			m.End()
		}(seed)
	}
	wg.Wait()
	close(errc)
	for err := range errc {
		t.Error(err)
	}
}
