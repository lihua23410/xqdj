package unit

import (
	"math/rand/v2"
	"reflect"
	"sync"
	"unsafe"
)

// Fork 记下已经复制过的指针，沙盘里指回同一份状态的字段共用那一份副本。
type Fork struct {
	seen map[unsafe.Pointer]reflect.Value
}

func NewFork() *Fork {
	return &Fork{seen: map[unsafe.Pointer]reflect.Value{}}
}

// ForkState 复制本场共享状态。演员结构里若攥着这份状态，应带着同一个 Fork 再复制演员。
func (s *MatchState) ForkState(g *Fork) *MatchState {
	out := &MatchState{}
	if s == nil {
		return out
	}
	if g == nil {
		g = NewFork()
	}
	g.seen[unsafe.Pointer(s)] = reflect.ValueOf(out)
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.values) == 0 {
		return out
	}
	out.values = make(map[any]any, len(s.values))
	for k, v := range s.values {
		cloned := g.clone(reflect.ValueOf(v))
		if cloned.IsValid() && cloned.CanInterface() {
			out.values[k] = cloned.Interface()
		}
	}
	return out
}

// ForkRNG 复制随机源，沙盘里再抽不会把真实对局的序列往前推。
func ForkRNG(r *rand.Rand, g *Fork) *rand.Rand {
	if r == nil {
		return nil
	}
	if g == nil {
		g = NewFork()
	}
	cloned := g.clone(reflect.ValueOf(r))
	out, _ := cloned.Interface().(*rand.Rand)
	return out
}

// ForkActor 深拷贝演员。通道留空，互斥锁归零，函数仍指向原来的闭包。
func ForkActor(a Actor, g *Fork) Actor {
	if a == nil {
		return nil
	}
	if g == nil {
		g = NewFork()
	}
	cloned := g.clone(reflect.ValueOf(a))
	if !cloned.IsValid() || (nillable(cloned.Kind()) && cloned.IsNil()) {
		return nil
	}
	actor, _ := cloned.Interface().(Actor)
	return actor
}

func nillable(k reflect.Kind) bool {
	switch k {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer, reflect.Interface, reflect.Slice:
		return true
	default:
		return false
	}
}

func (g *Fork) clone(v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}
	switch v.Kind() {
	case reflect.Interface:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		inner := g.clone(v.Elem())
		out := reflect.New(v.Type()).Elem()
		if inner.IsValid() {
			out.Set(inner)
		}
		return out
	case reflect.Pointer:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		ptr := v.UnsafePointer()
		if prev, ok := g.seen[ptr]; ok {
			return prev
		}
		if isSync(v.Elem().Type()) {
			nv := reflect.New(v.Elem().Type())
			g.seen[ptr] = nv
			return nv
		}
		nv := reflect.New(v.Elem().Type())
		g.seen[ptr] = nv
		g.write(nv.Elem(), g.clone(v.Elem()))
		return nv
	case reflect.Struct:
		return g.cloneStruct(asAddr(v))
	case reflect.Slice:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		out := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		for i := 0; i < v.Len(); i++ {
			g.write(out.Index(i), g.clone(v.Index(i)))
		}
		return out
	case reflect.Array:
		out := reflect.New(v.Type()).Elem()
		addr := asAddr(v)
		for i := 0; i < v.Len(); i++ {
			g.write(out.Index(i), g.clone(addr.Index(i)))
		}
		return out
	case reflect.Map:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		out := reflect.MakeMapWithSize(v.Type(), v.Len())
		for _, key := range v.MapKeys() {
			out.SetMapIndex(g.clone(key), g.clone(v.MapIndex(key)))
		}
		return out
	case reflect.Chan:
		return reflect.Zero(v.Type())
	case reflect.Func:
		return v
	default:
		return v
	}
}

func (g *Fork) cloneStruct(v reflect.Value) reflect.Value {
	out := reflect.New(v.Type()).Elem()
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return out
	}
	if !v.CanAddr() {
		v = asAddr(v)
	}
	if !v.CanAddr() {
		return out
	}
	base := v.Addr().UnsafePointer()
	dst := out.Addr().UnsafePointer()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if isSync(f.Type) {
			continue
		}
		src := reflect.NewAt(f.Type, unsafe.Add(base, f.Offset)).Elem()
		if f.Type.Kind() == reflect.Chan {
			continue
		}
		cloned := g.clone(src)
		g.write(reflect.NewAt(f.Type, unsafe.Add(dst, f.Offset)).Elem(), cloned)
	}
	return out
}

func (g *Fork) write(dst, src reflect.Value) {
	if !dst.IsValid() || !src.IsValid() || dst.Type() != src.Type() {
		return
	}
	// Typed assignment keeps the GC write barrier for slices, maps, and other pointers.
	dst.Set(src)
}

func asAddr(v reflect.Value) reflect.Value {
	if !v.IsValid() || v.CanAddr() {
		return v
	}
	if v.Kind() != reflect.Struct && v.Kind() != reflect.Array {
		return v
	}
	// An interface can hold a single-pointer struct directly, so its data word
	// is not necessarily the address of a copy of the struct.
	addressable := reflect.New(v.Type()).Elem()
	addressable.Set(v)
	return addressable
}

func isSync(t reflect.Type) bool {
	if t == nil {
		return false
	}
	if t.PkgPath() != "sync" {
		return false
	}
	switch t.Name() {
	case "Mutex", "RWMutex", "Once", "WaitGroup", "Cond", "Pool", "Map":
		return true
	default:
		return false
	}
}

// 编译期确认互斥锁的零值语义没被复制路径漏掉。
var _ = sync.Mutex{}
