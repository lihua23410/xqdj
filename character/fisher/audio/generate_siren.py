"""Generate the original police siren embedded by the fisher pack."""

from array import array
from math import cos, sin, tau
from pathlib import Path
import sys
import wave


RATE = 24000
DURATION = 1.6
OUTPUT = Path(__file__).resolve().parents[1] / "fx" / "police-siren.wav"


def main():
    samples = array("h")
    phase = 0.0
    for index in range(round(RATE * DURATION)):
        t = index / RATE
        sweep = (1 - cos(tau * t / 0.72)) / 2
        phase += tau * (650 + 520 * sweep) / RATE
        envelope = min(1, t / 0.055, (DURATION - t) / 0.18)
        tone = sin(phase) + 0.22 * sin(phase * 2) + 0.08 * sin(phase * 3)
        samples.append(round(9500 * envelope * tone))
    if sys.byteorder == "big":
        samples.byteswap()
    with wave.open(str(OUTPUT), "wb") as out:
        out.setparams((1, 2, RATE, 0, "NONE", "not compressed"))
        out.writeframes(samples.tobytes())


if __name__ == "__main__":
    main()
