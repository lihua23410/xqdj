"""Original glass/tar synthesis mixed with the documented CC0 fire recordings."""
from pathlib import Path
import hashlib
import json
import math
import re
import subprocess
import tempfile
import wave
import numpy as np

RATE = 44100
PACK = Path(__file__).resolve().parents[1]
SOURCE = PACK / 'audio' / 'source'


def read_audio(path):
    raw = subprocess.run(['ffmpeg', '-v', 'error', '-i', str(path), '-ac', '1', '-ar', str(RATE), '-f', 'f32le', '-'], capture_output=True, check=True).stdout
    return np.frombuffer(raw, dtype='<f4').astype(np.float64)


def noise(rng, n, low, high):
    f = np.fft.rfftfreq(n, 1 / RATE)
    weight = (f / low) ** 3 / (1 + (f / low) ** 3) / (1 + (f / high) ** 6)
    signal = np.fft.irfft(np.fft.rfft(rng.normal(size=n)) * weight, n=n)
    return signal / max(np.std(signal), 1e-8)


def glass_tar(duration, seed, viscous):
    rng = np.random.default_rng(seed)
    t = np.arange(round(RATE * duration)) / RATE
    result = noise(rng, len(t), 1600, 7300) * np.exp(-t / .012) * .22
    # Inharmonic glass modes and tiny shards, rather than a metallic bell.
    for onset, level in [(0, .12), (.021, .065), (.063, .035), (.119, .02), (.187, .014)]:
        age = np.maximum(0, t - onset)
        envelope = (t >= onset) * (1 - np.exp(-age / .0007))
        for f in [1379, 2381, 3587, 4969, 6911]:
            result += level * envelope * np.exp(-age / (.014 + rng.random() * .025)) * np.sin(2 * math.pi * f * age + rng.random() * 5)
    if viscous:
        # Damped low liquid pulses, a thick splash and falling globs of tar.
        for i, onset in enumerate([.025, .073, .141, .233, .337]):
            age = np.maximum(0, t - onset)
            envelope = (t >= onset) * (1 - np.exp(-age / .004)) * np.exp(-age / (.048 - i * .004))
            phase = 2 * math.pi * (90 * age + 155 * .022 * (1 - np.exp(-age / .022)))
            result += envelope * (np.sin(phase) * .20 + noise(rng, len(t), 95, 1450) * .10) / (1 + i * .23)
        result += noise(rng, len(t), 140, 2400) * np.maximum(0, np.sin(np.minimum(t / .44, 1) * math.pi)) ** 2 * .032
    result *= np.minimum(1, t / .0005) * np.minimum(1, np.maximum(0, (duration - t) / .07))
    return result - np.mean(result)


def write(path, signal):
    with wave.open(str(path), 'wb') as out:
        out.setnchannels(1); out.setsampwidth(2); out.setframerate(RATE)
        out.writeframes(np.rint(np.clip(signal, -1, 1) * 32767).astype('<i2').tobytes())


def measure(path):
    result = subprocess.run(['ffmpeg', '-hide_banner', '-i', str(path), '-af', 'ebur128=peak=true', '-f', 'null', '-'], capture_output=True, text=True, check=True)
    summary = result.stderr.rsplit('Summary:', 1)[-1]
    return float(re.search(r'I:\s*([-\d.]+)\s*LUFS', summary)[1]), float(re.search(r'Peak:\s*([-\d.]+)\s*dBFS', summary)[1])


def main():
    catch = read_audio(SOURCE / 'fire-catch.wav')
    crackle = read_audio(SOURCE / 'fire-crackle.wav')
    oil = glass_tar(.62, 5801, True)
    fire = glass_tar(.85, 5802, False)
    offset = round(.025 * RATE)
    n = min(len(catch), len(fire) - offset)
    fire[offset:offset+n] += catch[:n] / max(np.max(np.abs(catch)), 1e-8) * .32
    fire *= np.minimum(1, (len(fire) - np.arange(len(fire))) / (RATE * .07))
    signals = [('oil-break.wav', oil, -15.7, -1.05), ('fire-break.wav', fire, -15.7, -1.05)]
    for name, start in [('crackle-a.wav', .03), ('crackle-b.wav', 1.35)]:
        clip = crackle[round(start*RATE):round((start+1.15)*RATE)].copy()
        t = np.arange(len(clip)) / RATE
        clip *= np.minimum(1, t / .035) * np.minimum(1, (len(clip)/RATE - t) / .1)
        signals.append((name, clip, -23.5, -7.05))
    records = []
    with tempfile.TemporaryDirectory(prefix='arson-audio-') as temp:
        for name, signal, target, ceiling in signals:
            signal *= .7 / max(np.max(np.abs(signal)), 1e-8)
            raw = Path(temp) / name
            write(raw, signal)
            processed = Path(temp) / ('filtered-' + name)
            filters = 'lowpass=f=7600,acompressor=threshold=0.035:ratio=6:attack=0.1:release=35' if 'crackle' in name else 'lowpass=f=7600,acompressor=threshold=0.16:ratio=3:attack=0.6:release=45'
            subprocess.run(['ffmpeg', '-v', 'error', '-y', '-i', str(raw), '-af', filters, '-c:a', 'pcm_s16le', str(processed)], check=True)
            loudness, peak = measure(processed)
            gain = min(target - loudness, ceiling - peak)
            output = PACK / 'fx' / name
            subprocess.run(['ffmpeg', '-v', 'error', '-y', '-i', str(processed), '-af', f'volume={gain:.4f}dB', '-c:a', 'pcm_s16le', str(output)], check=True)
            loudness, peak = measure(output)
            if peak > ceiling + .1:
                raise RuntimeError(f'{name}: true peak {peak} exceeds {ceiling}')
            records.append({'file': name, 'duration_seconds': round(len(signal)/RATE, 4), 'sample_rate': RATE, 'channels': 1, 'integrated_lufs': loudness, 'true_peak_dbfs': peak,
                'source': 'Original modal glass and viscous-liquid synthesis (seed 5801)' if name == 'oil-break.wav' else 'Original modal glass (seed 5802) + CC0 Catching fire / themightyglider' if name == 'fire-break.wav' else 'CC0 Fire Crackling / AntumDeluge',
                'license': 'MIT' if name == 'oil-break.wav' else 'MIT synthesis; CC0 1.0 recording' if name == 'fire-break.wav' else 'CC0 1.0',
                'processing': f'44.1kHz mono PCM16; {filters}; fades; gain {gain:.4f}dB; target {target} LUFS, ceiling {ceiling} dBTP',
                'mode': 0, 'priority': 210 if 'break' in name else 230, 'volume': 1.0, 'sha256': hashlib.sha256(output.read_bytes()).hexdigest()})
    (PACK / 'fx' / 'audio-sources.json').write_text(json.dumps(records, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps(records, ensure_ascii=False, indent=2))

if __name__ == '__main__':
    main()
