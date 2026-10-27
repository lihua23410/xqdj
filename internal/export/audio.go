package export

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	unitpkg "xqdj/internal/unit"
	"xqdj/internal/web"
)

const audioRate = 48000

type audioEvent struct {
	Src            string  `json:"src"`
	Frame          int     `json:"frame"`
	Volume         float64 `json:"volume"`
	Loop           bool    `json:"loop"`
	Rate           float64 `json:"rate"`
	PreservesPitch *bool   `json:"preservesPitch"`
	Offset         float64 `json:"offset"`
	Stop           *int    `json:"stop"`
}

// mixAudio 把页面按帧记下的播放意图混成一条 WAV。没有事件时写等长静音。
func mixAudio(ctx context.Context, ffmpeg, eventsJSON, video, out string, totalFrames int) error {
	var events []audioEvent
	if strings.TrimSpace(eventsJSON) != "" {
		if err := json.Unmarshal([]byte(eventsJSON), &events); err != nil {
			return err
		}
	}
	samples := totalFrames * (audioRate / 60)
	mix := make([]float32, samples*2)
	if len(events) > 0 {
		if err := mixEvents(ctx, ffmpeg, events, mix); err != nil {
			return err
		}
	}
	// 多路声音叠加可能超过 0 dBFS，直接写 int16 会硬削波，
	// 听感就是发毛的“电音”。留一点余量，按峰值整体收一次。
	peak := 0.0
	for _, s := range mix {
		if a := math.Abs(float64(s)); a > peak {
			peak = a
		}
	}
	if peak > 0.98 {
		gain := float32(0.98 / peak)
		for i := range mix {
			mix[i] *= gain
		}
	}
	return writeWAV(out, mix)
}

func mixEvents(ctx context.Context, ffmpeg string, events []audioEvent, mix []float32) error {
	stopAt := make(map[int]int)
	for _, ev := range events {
		if ev.Stop != nil {
			stopAt[*ev.Stop] = ev.Frame
		}
	}
	cache := make(map[string][]float32)
	for i, ev := range events {
		if ev.Stop != nil || ev.Src == "" {
			continue
		}
		key := normalizeSrc(ev.Src)
		decoded, ok := cache[key]
		if !ok {
			buf, err := decodeAudio(ctx, ffmpeg, key, ev.Rate, ev.PreservesPitch)
			if err != nil {
				return fmt.Errorf("%s: %w", key, err)
			}
			decoded = buf
			cache[key] = decoded
		}
		start := ev.Frame * (audioRate / 60)
		if start < 0 || start >= len(mix)/2 {
			continue
		}
		skip := int(ev.Offset * audioRate)
		if skip < 0 {
			skip = 0
		}
		if skip*2 >= len(decoded) {
			continue
		}
		src := decoded[skip*2:]
		vol := ev.Volume
		if math.IsNaN(vol) || vol < 0 {
			vol = 0
		}
		if vol > 1 {
			vol = 1
		}
		limit := len(mix)/2 - start
		// pause 事件对非循环音同样生效：浏览器里前一声会被截断，
		// 导出混音不能再让它一直铺到底，否则叠音会听成电音。
		if stop, ok := stopAt[i]; ok {
			if n := (stop - ev.Frame) * (audioRate / 60); n < limit {
				limit = n
			}
		}
		if !ev.Loop && len(src)/2 < limit {
			limit = len(src) / 2
		}
		if limit <= 0 {
			continue
		}
		for n := 0; n < limit; n++ {
			idx := (n * 2) % len(src)
			mix[(start+n)*2] += src[idx] * float32(vol)
			mix[(start+n)*2+1] += src[idx+1] * float32(vol)
		}
	}
	return nil
}

func normalizeSrc(raw string) string {
	if u, err := url.Parse(raw); err == nil && u.Path != "" {
		raw = u.Path
	}
	raw = strings.TrimPrefix(path.Clean("/"+raw), "/")
	raw = strings.TrimPrefix(raw, "ball/")
	return raw
}

func decodeAudio(ctx context.Context, ffmpeg, name string, rate float64, preservesPitch *bool) ([]float32, error) {
	data, err := readAudioFile(name)
	if err != nil {
		return nil, err
	}
	args := []string{"-hide_banner", "-loglevel", "error", "-i", "-", "-f", "f32le", "-acodec", "pcm_f32le", "-ac", "2", "-ar", strconv.Itoa(audioRate)}
	if rate > 0 && math.Abs(rate-1) > 0.001 {
		keepPitch := preservesPitch == nil || *preservesPitch
		if keepPitch && rate >= 0.5 && rate <= 2 {
			args = append(args, "-filter:a", "atempo="+strconv.FormatFloat(rate, 'f', 3, 64))
		}
	}
	args = append(args, "-")
	cmd := exec.CommandContext(ctx, ffmpeg, args...)
	cmd.Stdin = bytes.NewReader(data)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg 解码: %w: %s", err, strings.TrimSpace(errBuf.String()))
	}
	if len(out)%8 != 0 {
		out = out[:len(out)-(len(out)%8)]
	}
	buf := make([]float32, len(out)/4)
	for i := range buf {
		buf[i] = math.Float32frombits(binary.LittleEndian.Uint32(out[i*4:]))
	}
	return buf, nil
}

func readAudioFile(name string) ([]byte, error) {
	if strings.HasPrefix(name, "sfx/") {
		return web.FS.ReadFile(name)
	}
	return readBallFile(name)
}

func readBallFile(name string) ([]byte, error) {
	f, err := unitpkg.BallFS().Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

func writeWAV(out string, samples []float32) error {
	dataLen := len(samples) * 2
	var b bytes.Buffer
	b.WriteString("RIFF")
	_ = binary.Write(&b, binary.LittleEndian, uint32(36+dataLen))
	b.WriteString("WAVEfmt ")
	_ = binary.Write(&b, binary.LittleEndian, uint32(16))
	_ = binary.Write(&b, binary.LittleEndian, uint16(1))
	_ = binary.Write(&b, binary.LittleEndian, uint16(2))
	_ = binary.Write(&b, binary.LittleEndian, uint32(audioRate))
	_ = binary.Write(&b, binary.LittleEndian, uint32(audioRate*2*2))
	_ = binary.Write(&b, binary.LittleEndian, uint16(4))
	_ = binary.Write(&b, binary.LittleEndian, uint16(16))
	b.WriteString("data")
	_ = binary.Write(&b, binary.LittleEndian, uint32(dataLen))
	for _, s := range samples {
		if s > 1 {
			s = 1
		} else if s < -1 {
			s = -1
		}
		_ = binary.Write(&b, binary.LittleEndian, int16(s*32767))
	}
	return os.WriteFile(out, b.Bytes(), 0o644)
}
