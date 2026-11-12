"""Generate original, deterministic shield sounds; no external samples required."""

from pathlib import Path
import hashlib
import json
import math
import re
import subprocess
import wave

import numpy as np


RATE = 44100
PACK = Path(__file__).resolve().parents[1]
DRAFT = PACK.parents[1] / ".temp" / "warden-audio-source"


def filtered_noise(rng, count, low, high):
    noise = rng.standard_normal(count)
    freq = np.fft.rfftfreq(count, 1 / RATE)
    mask = (freq / np.maximum(low, 1)) ** 3
    mask /= 1 + mask
    mask /= 1 + (freq / high) ** 6
    result = np.fft.irfft(np.fft.rfft(noise) * mask, n=count)
    return result / max(np.std(result), 1e-9)


def shield_sound():
    duration = 0.28
    rng = np.random.default_rng(4816)
    t = np.arange(round(RATE * duration)) / RATE
    signal = np.zeros_like(t)
    modes = [233, 479, 857, 1391, 2137, 3181, 4621, 6317]
    for index, frequency in enumerate(modes):
        decay = 0.105 / (1 + index * 0.18)
        attack = 1 - np.exp(-t / 0.0012)
        signal += 0.16 * attack * np.exp(-t / decay) * np.sin(2 * math.pi * frequency * t + index * 0.37)
    # A brief low plate impact gives the break weight without a long boom.
    phase = 2 * math.pi * (175 * t + 42 * 0.018 * (1 - np.exp(-t / 0.018)))
    signal += 0.48 * np.sin(phase) * (1 - np.exp(-t / 0.0008)) * np.exp(-t / 0.028)
    noise = filtered_noise(rng, len(t), 620, 6200)
    signal += noise * 0.28 * (1 - np.exp(-t / 0.0005)) * np.exp(-t / 0.021)
    # Uneven, damped fragment clicks keep the tail granular rather than tonal.
    onsets = [0.009, 0.019, 0.035, 0.056, 0.083, 0.119]
    for index, onset in enumerate(onsets):
        age = np.maximum(0, t - onset)
        envelope = (t >= onset) * (1 - np.exp(-age / 0.0007)) * np.exp(-age / 0.013)
        grit = filtered_noise(rng, len(t), 1800, 7200)
        signal += grit * envelope * 0.14 / (1 + index * 0.28)
    signal -= np.mean(signal)
    signal *= np.minimum(1, t / 0.0006)
    signal *= np.minimum(1, np.maximum(0, (duration - t) / 0.024)) ** 2
    signal *= 10 ** (-2 / 20) / max(np.max(np.abs(signal)), 1e-9)
    return signal


def write_wav(path, signal):
    with wave.open(str(path), "wb") as output:
        output.setnchannels(1)
        output.setsampwidth(2)
        output.setframerate(RATE)
        output.writeframes(np.rint(np.clip(signal, -1, 1) * 32767).astype("<i2").tobytes())


def measure(path):
    result = subprocess.run(
        ["ffmpeg", "-hide_banner", "-i", str(path), "-filter_complex", "ebur128=peak=true", "-f", "null", "-"],
        capture_output=True, text=True, check=True,
    )
    summary = result.stderr.rsplit("Summary:", 1)[-1]
    peak = float(re.search(r"Peak:\s*([-\d.]+)\s*dBFS", summary).group(1))
    loudness = float(re.search(r"I:\s*([-\d.]+)\s*LUFS", summary).group(1))
    return loudness, peak


def main():
    DRAFT.mkdir(parents=True, exist_ok=True)
    records = []
    for name in ["shatter-full.wav"]:
        raw = DRAFT / name
        output = PACK / "fx" / name
        write_wav(raw, shield_sound())
        _, peak = measure(raw)
        gain = -1.05 - peak
        subprocess.run(
            ["ffmpeg", "-hide_banner", "-loglevel", "error", "-y", "-i", str(raw), "-af", f"volume={gain:.3f}dB", "-c:a", "pcm_s16le", str(output)],
            check=True,
        )
        loudness, final_peak = measure(output)
        if final_peak > -1.0:
            raise RuntimeError(f"True peak limit exceeded: {name}: {final_peak}")
        with wave.open(str(output), "rb") as audio:
            duration = audio.getnframes() / audio.getframerate()
        records.append({
            "file": name, "source": "Original deterministic modal/noise synthesis; no external samples",
            "license": "MIT (repository LICENSE)", "seed": 4816,
            "duration_seconds": duration, "sample_rate": RATE, "channels": 1,
            "integrated_lufs": loudness, "true_peak_dbfs": final_peak,
            "normalization": "Short effect: true peak target -1 dBFS; integrated LUFS is not meaningful",
            "mode": 2, "priority": 0, "volume": 1.0,
            "sha256": hashlib.sha256(output.read_bytes()).hexdigest(),
        })
    (PACK / "fx" / "audio-sources.json").write_text(json.dumps(records, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(records, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
