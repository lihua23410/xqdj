"""Original deterministic surgical foley. No recordings or external samples."""
from pathlib import Path
import math
import random
import struct
import wave

RATE = 44100
OUT = Path(__file__).resolve().parent.parent / "fx"

def render(name, duration, kind):
    rng = random.Random(731 + kind)
    samples = []
    low = 0.0
    for i in range(int(RATE * duration)):
        t = i / RATE
        noise = rng.uniform(-1, 1)
        low += .16 * (noise - low)
        env = min(1, t / .012) * (1 - t / duration) ** 2
        if kind == 0:
            value = (.55 * low + .09 * math.sin(2 * math.pi * (1700 - 1800 * t) * t)) * env
        elif kind == 1:
            value = (.5 * low + .08 * math.sin(2 * math.pi * 1100 * t) * math.exp(-t * 55)) * env
        else:
            value = (low * .7 + math.sin(2 * math.pi * 105 * t) * .13) * env
        samples.append(value)
    peak = max(abs(v) for v in samples) or 1
    gain = 10 ** (-8 / 20) / peak
    with wave.open(str(OUT / name), "wb") as dst:
        dst.setparams((1, 2, RATE, 0, "NONE", "not compressed"))
        dst.writeframes(b"".join(struct.pack("<h", round(v * gain * 32767)) for v in samples))

if __name__ == "__main__":
    render("cut.wav", .24, 0)
    render("stitch.wav", .13, 1)
    render("burst.wav", .38, 2)
