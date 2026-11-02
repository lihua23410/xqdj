package sim

import (
	"bytes"
	"encoding/json"
	"testing"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func TestArsonistSurfaceSurvivesSnapshotPauseAndRecording(t *testing.T) {
	m := NewMatchSeeded(41)
	m.SetSlot(0, character.KindArsonist)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	defer m.End()
	var packet []byte
	for i := 0; i < 600; i++ {
		m.Tick()
		raw := m.SnapshotJSON()
		var snap struct {
			Effects []unitpkg.FX `json:"effects"`
		}
		if err := json.Unmarshal(raw, &snap); err != nil {
			t.Fatal(err)
		}
		for _, fx := range snap.Effects {
			if fx.Name != "arson-surface" {
				continue
			}
			var surface struct {
				Chunks []json.RawMessage `json:"chunks"`
			}
			if err := json.Unmarshal(fx.Data, &surface); err != nil {
				t.Fatalf("surface must be forwarded as a JSON object: %v", err)
			}
			if len(surface.Chunks) > 0 {
				packet = raw
				break
			}
		}
		if packet != nil {
			break
		}
	}
	if packet == nil {
		t.Fatal("no oil surface reached the client after the first bottle")
	}

	m.recordSnapshot(packet)
	recorded := m.Recording()
	if len(recorded.Frames) != 1 || !bytes.Equal(recorded.Frames[0], packet) {
		t.Fatal("recording lost the surface data")
	}
	m.Pause()
	var before, paused struct {
		Effects json.RawMessage `json:"effects"`
	}
	if err := json.Unmarshal(packet, &before); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(m.SnapshotJSON(), &paused); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before.Effects, paused.Effects) {
		t.Fatal("pause lost the surface data")
	}
}
