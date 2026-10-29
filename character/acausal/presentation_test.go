package 非因果

import "testing"

func TestRewindPresentationEnablesManagedAudio(t *testing.T) {
	p, err := RewindPresentation(&IntroFilm{Death: 60, Frames: []IntroFrame{{T: 0}}})
	if err != nil {
		t.Fatal(err)
	}
	if !p.Audio || p.Pack != KindAcausal || p.Type != "rewind" || p.Duration != 7 {
		t.Fatalf("rewind must enable audio for its full duration: %+v", p)
	}
}
