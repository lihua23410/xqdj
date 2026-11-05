package unit

import "sync"

// MatchState 只属于一场对局。各包定义自己的私有状态类型，引擎只传递作用域。
// 零值可用；使用后不能复制。每次重开都换一份，先等旧演员退出再释放。
type MatchState struct {
	mu     sync.Mutex
	values map[any]any
}

type stateKey[T any] struct{}

// StateFor 返回本场唯一的 T。T 应是包内私有类型，避免不同包共用同一个内置类型。
// 初始化只发生一次；T 内部字段的并发读写仍由该包自己保护。
func StateFor[T any](scope *MatchState) *T {
	if scope == nil {
		panic("unit: missing match state")
	}
	scope.mu.Lock()
	defer scope.mu.Unlock()
	if scope.values == nil {
		scope.values = make(map[any]any)
	}
	key := stateKey[T]{}
	if value, ok := scope.values[key]; ok {
		return value.(*T)
	}
	value := new(T)
	scope.values[key] = value
	return value
}
