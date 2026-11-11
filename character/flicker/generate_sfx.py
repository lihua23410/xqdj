"""Original deterministic synthesized effects; no recordings or third-party assets.
Run: python character/flicker/generate_sfx.py
"""
import math
from pathlib import Path
import random
import struct
import wave

RATE = 48000
ROOT = Path(__file__).parent / "fx"

def render(name, duration, start, end, noise, peak):
    rng = random.Random(2077)
    phase = smooth = 0.0
    samples = []
    for i in range(int(duration * RATE)):
        t = i / RATE
        u = t / duration
        phase += 2 * math.pi * (end + (start - end) * (1 - u) ** 3) / RATE
        smooth += .13 * (rng.uniform(-1, 1) - smooth)
        envelope = min(1, t / .003) * (1 - u) ** 2
        tone = math.sin(phase) * .65 + math.sin(phase * 1.498) * .16
        samples.append((tone * (1 - noise) + smooth * noise * 3) * envelope)
    gain = peak / max(abs(v) for v in samples)
    with wave.open(str(ROOT / f"{name}.wav"), "wb") as out:
        out.setparams((1, 2, RATE, 0, "NONE", "not compressed"))
        out.writeframes(b"".join(struct.pack("<h", round(v * gain * 32767)) for v in samples))

render("reflex", .18, 2300, 180, .65, .65)
render("shot", .085, 1500, 330, .28, .55)
render("overdrive", .14, 1150, 140, .4, .65)
render("armed", .065, 1100, 1700, .04, .35)
