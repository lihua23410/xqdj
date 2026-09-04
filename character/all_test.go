package character

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAllGoListsEveryPack(t *testing.T) {
	src, err := os.ReadFile("all.go")
	if err != nil {
		t.Fatalf("read all.go: %v (run go generate ./character)", err)
	}
	ents, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	var missing []string
	for _, e := range ents {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		ok, err := dirHasGo(e.Name())
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			continue
		}
		want := `"xqdj/character/` + e.Name() + `"`
		if !strings.Contains(string(src), want) {
			missing = append(missing, e.Name())
		}
	}
	if len(missing) > 0 {
		t.Fatalf("all.go missing packs %v; run go generate ./character", missing)
	}
}

func dirHasGo(dir string) (bool, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			return true, nil
		}
	}
	return false, nil
}

func TestAllGoNoStaleImports(t *testing.T) {
	src, err := os.ReadFile("all.go")
	if err != nil {
		t.Skip("no all.go")
	}
	const prefix = `"xqdj/character/`
	rest := string(src)
	for {
		i := strings.Index(rest, prefix)
		if i < 0 {
			break
		}
		rest = rest[i+len(prefix):]
		j := strings.Index(rest, `"`)
		if j < 0 {
			break
		}
		dir := rest[:j]
		rest = rest[j+1:]
		goFiles, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for _, f := range goFiles {
			if !strings.HasSuffix(f, "_test.go") {
				n++
			}
		}
		if n == 0 {
			t.Fatalf("all.go imports %s but that dir has no .go files; run go generate ./character", dir)
		}
	}
}
