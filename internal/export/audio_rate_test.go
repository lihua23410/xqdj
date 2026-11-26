package export

import (
	"bytes"
	"context"
	"encoding/binary"
	"math"
	"testing"
	"time"

	_ "xqdj/character/wolf"
)

func TestAudioCacheKeyKeepsRateAndPitch(t *testing.T) {
	name := "狼人/fx/animals.mp3"
	plain := audioCacheKey(name, 1, true)
	faster := audioCacheKey(name, 2.2, true)
	pitched := audioCacheKey(name, 2.2, false)
	if plain == faster || faster == pitched || plain == pitched {
		t.Fatalf("cache keys collided: %q %q %q", plain, faster, pitched)
	}
	if audioCacheKey(name, eventRate(audioEvent{Rate: 0}), true) != plain {
		t.Fatal("missing rate should share the 1x entry")
	}
}

func TestSampleSkipUsesMediaTime(t *testing.T) {
	if got, want := sampleSkip(2, 2), audioRate; got != want {
		t.Fatalf("skip=%d want %d", got, want)
	}
	if sampleSkip(0, 2) != 0 || sampleSkip(-1, 1.5) != 0 {
		t.Fatal("non-positive offset should not skip")
	}
}

func TestRateFilterCoversFastAndRepitch(t *testing.T) {
	chain := atempoChain(2.2)
	if chain != "atempo=2,atempo=1.100000" {
		t.Fatalf("atempo chain: %s", chain)
	}
	if got := rateFilter(2.2, false); got != "aformat=sample_rates=48000,asetrate=105600,aresample=48000" {
		t.Fatalf("repitch filter: %s", got)
	}
	if rateFilter(1, true) != "" || rateFilter(1, false) != "" {
		t.Fatal("unity rate should not filter")
	}
}

func TestDecodeRateChangesSpeedAndPitch(t *testing.T) {
	ffmpeg := findFFmpeg()
	if ffmpeg == "" {
		t.Skip("no ffmpeg")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tone := sineWAV(440, 2)
	keep := true
	shift := false
	normal, err := decodePCM(ctx, ffmpeg, tone, 1, &keep)
	if err != nil {
		t.Fatal(err)
	}
	fast, err := decodePCM(ctx, ffmpeg, tone, 2.4, &keep)
	if err != nil {
		t.Fatal(err)
	}
	repitch, err := decodePCM(ctx, ffmpeg, tone, 2, &shift)
	if err != nil {
		t.Fatal(err)
	}
	if ratio := pcmSeconds(fast) / pcmSeconds(normal); ratio > 0.6 || ratio < 0.3 {
		t.Fatalf("2.4x keep-pitch duration ratio %v", ratio)
	}
	if hz := dominantHz(fast); math.Abs(hz-440) > 40 {
		t.Fatalf("keep-pitch frequency %v Hz", hz)
	}
	if ratio := pcmSeconds(repitch) / pcmSeconds(normal); ratio > 0.65 || ratio < 0.35 {
		t.Fatalf("2x repitch duration ratio %v", ratio)
	}
	if hz := dominantHz(repitch); math.Abs(hz-880) > 60 {
		t.Fatalf("repitch frequency %v Hz", hz)
	}
}

func TestHowlRatesDoNotSharePCM(t *testing.T) {
	ffmpeg := findFFmpeg()
	if ffmpeg == "" {
		t.Skip("no ffmpeg")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	src := "狼人/fx/animals.mp3"
	keep := true
	slow, err := decodeAudio(ctx, ffmpeg, src, 1, &keep)
	if err != nil {
		t.Fatal(err)
	}
	fast, err := decodeAudio(ctx, ffmpeg, src, 2.2, &keep)
	if err != nil {
		t.Fatal(err)
	}
	if len(slow) < audioRate || len(fast) >= len(slow)*3/4 {
		t.Fatalf("howl pcm slow=%d fast=%d", len(slow), len(fast))
	}
	gap := len(slow)/2/(audioRate/60) + 30
	total := gap + len(slow)/2/(audioRate/60) + 30
	mix := make([]float32, total*(audioRate/60)*2)
	err = mixEvents(ctx, ffmpeg, []audioEvent{
		{Src: src, Frame: 0, Volume: 1, Rate: 1, PreservesPitch: &keep},
		{Src: src, Frame: gap, Volume: 1, Rate: 2.2, PreservesPitch: &keep},
	}, mix)
	if err != nil {
		t.Fatal(err)
	}
	first := audibleSamples(mix, 0, gap*(audioRate/60))
	second := audibleSamples(mix, gap*(audioRate/60), len(mix)/2)
	if first == 0 || second == 0 {
		t.Fatalf("silent howl first=%d second=%d", first, second)
	}
	if second > first*3/4 {
		t.Fatalf("cached 1x howl into the 2.2x slot: first=%d second=%d", first, second)
	}
}

func sineWAV(freq, seconds float64) []byte {
	n := int(seconds * audioRate)
	var b bytes.Buffer
	dataLen := n * 2 * 2
	b.WriteString("RIFF")
	_ = binary.Write(&b, binary.LittleEndian, uint32(36+dataLen))
	b.WriteString("WAVEfmt ")
	_ = binary.Write(&b, binary.LittleEndian, uint32(16))
	_ = binary.Write(&b, binary.LittleEndian, uint16(1))
	_ = binary.Write(&b, binary.LittleEndian, uint16(2))
	_ = binary.Write(&b, binary.LittleEndian, uint32(audioRate))
	_ = binary.Write(&b, binary.LittleEndian, uint32(audioRate*4))
	_ = binary.Write(&b, binary.LittleEndian, uint16(4))
	_ = binary.Write(&b, binary.LittleEndian, uint16(16))
	b.WriteString("data")
	_ = binary.Write(&b, binary.LittleEndian, uint32(dataLen))
	for i := 0; i < n; i++ {
		s := int16(math.Sin(2*math.Pi*freq*float64(i)/audioRate) * 16000)
		_ = binary.Write(&b, binary.LittleEndian, s)
		_ = binary.Write(&b, binary.LittleEndian, s)
	}
	return b.Bytes()
}

func pcmSeconds(buf []float32) float64 {
	return float64(len(buf)/2) / audioRate
}

func dominantHz(buf []float32) float64 {
	if len(buf) < 4 {
		return 0
	}
	// 丢掉 atempo 前后的淡入淡出，只量中间。
	from := len(buf) / 8
	to := len(buf) * 7 / 8
	if from%2 != 0 {
		from++
	}
	crossings := 0
	prev := buf[from]
	samples := 0
	for i := from + 2; i < to; i += 2 {
		s := buf[i]
		if (prev < 0 && s >= 0) || (prev >= 0 && s < 0) {
			crossings++
		}
		prev = s
		samples++
	}
	if samples == 0 {
		return 0
	}
	return float64(crossings) / 2 / (float64(samples) / audioRate)
}

func audibleSamples(mix []float32, start, end int) int {
	if start < 0 {
		start = 0
	}
	if end > len(mix)/2 {
		end = len(mix) / 2
	}
	last := 0
	for i := start; i < end; i++ {
		if math.Abs(float64(mix[i*2])) > 0.02 || math.Abs(float64(mix[i*2+1])) > 0.02 {
			last = i - start + 1
		}
	}
	return last
}
