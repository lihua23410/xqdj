package unit

import (
	"io/fs"
	"path"
	"sort"
	"strings"
)

type Pack struct {
	Name     string
	FS       fs.FS
	factions []FactionLook
}

type PackInfo struct {
	Base     string        `json:"base"`
	Files    []string      `json:"files,omitempty"`
	Factions []FactionLook `json:"factions,omitempty"`
}

// FactionLook 是派系图标/颜色。Icon 用绝对路径，和状态图一样：/ball/<Kind>/faction/qing.png
type FactionLook struct {
	ID    string `json:"id"`
	Icon  string `json:"icon,omitempty"`
	Color string `json:"color,omitempty"`
	File  string `json:"-"`
}

var packs = map[string]*Pack{}

func NewPack(name string, assets fs.FS) *Pack {
	if name == "" {
		panic("unit: empty pack name")
	}
	if _, ok := packs[name]; ok {
		panic("unit: duplicate pack " + name)
	}
	p := &Pack{Name: name, FS: assets}
	packs[name] = p
	return p
}

func (p *Pack) Register(s Spec, fn func(SpawnInfo) Actor) {
	s.Look.Base = "/ball/" + p.Name
	Register(s, fn)
}

// RegisterFactions 登记本包派系图。File 相对包根；引擎拼成 /ball/<Kind>/… 发给页面。
func (p *Pack) RegisterFactions(list []FactionLook) {
	if p == nil {
		return
	}
	base := "/ball/" + p.Name
	out := make([]FactionLook, 0, len(list))
	for _, item := range list {
		if item.ID == "" {
			continue
		}
		icon := item.Icon
		if icon == "" && item.File != "" {
			icon = path.Join(base, item.File)
		}
		out = append(out, FactionLook{ID: item.ID, Icon: icon, Color: item.Color})
	}
	p.factions = out
}

func Packs() map[string]PackInfo {
	out := make(map[string]PackInfo, len(packs))
	for name, p := range packs {
		info := PackInfo{Base: "/ball/" + name, Files: listPackFiles(p.FS), Factions: p.factions}
		out[name] = info
	}
	return out
}

func listPackFiles(fsys fs.FS) []string {
	if fsys == nil {
		return nil
	}
	var files []string
	_ = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || p == "." {
			return err
		}
		base := path.Base(p)
		if strings.HasPrefix(base, ".") || strings.HasPrefix(base, "_") {
			return nil
		}
		files = append(files, path.Clean(p))
		return nil
	})
	sort.Strings(files)
	return files
}

// BallFS 把各包的 embed 挂成 /<Kind>/fx/... 给 HTTP 用。
func BallFS() fs.FS {
	return ballFS{}
}

type ballFS struct{}

func (ballFS) Open(name string) (fs.File, error) {
	name = strings.TrimPrefix(path.Clean("/"+name), "/")
	if name == "." || name == "" {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	packName, rest, ok := strings.Cut(name, "/")
	if !ok || packName == "" || rest == "" || rest == "." {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	p := packs[packName]
	if p == nil || p.FS == nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return p.FS.Open(rest)
}
