package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"testing/fstest"
)

func TestPairingRosterReadsCurrentImports(t *testing.T) {
	source := fstest.MapFS{
		"all.go": {Data: []byte(`package character
import (
	"xqdj/character/archer"
	"xqdj/character/dummy"
)`)},
		"archer/names.go": {Data: []byte(`package 弓箭; const KindArcher = "弓箭"`)},
		"archer/register.go": {Data: []byte(`package 弓箭
func init() {
	p.Register(unit.Spec{Kind: KindArcher, Fighter: true}, spawn)
	p.Register(unit.Spec{Kind: "箭矢"}, spawnArrow)
}`)},
		"dummy/register.go": {Data: []byte(`package 靶子
func init() { p.Register(unit.Spec{Kind: "靶子", Fighter: true}, spawn) }`)},
		"echo/register.go": {Data: []byte(`package 回声
func init() { p.Register(unit.Spec{Kind: "回声", Fighter: true}, spawn) }`)},
	}
	handler := PairingRoster(source)
	check := func(want ...string) {
		t.Helper()
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/pairing-roster", nil))
		if res.Code != http.StatusOK || res.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("unexpected response: %d %v %s", res.Code, res.Header(), res.Body.String())
		}
		var got []string
		if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil || !slices.Equal(got, want) {
			t.Fatalf("got %v (%v), want %v", got, err, want)
		}
	}
	check("弓箭")
	source["all.go"].Data = []byte(`package character; import "xqdj/character/echo"`)
	check("回声")
}
