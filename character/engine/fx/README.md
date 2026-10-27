# 内燃机 FX 资源

## attack.wav

内燃机普通攻击的抢占音效。

- 原始文件：`tmp/fire-sfx/2026-10-27/fire-flare-ignition-01.wav`
- 处理：`ffmpeg -filter:a atempo=2.0`（2 倍速，保留音高），44.1kHz / 16-bit / stereo
- 来源：OpenGameArt — *Flare ignition* — qubodup
  https://opengameart.org/content/flare-ignition
- 许可：CC0 1.0 Universal，全文见同目录 `LICENSE-CC0-1.0.txt`
- 署名：不强制；可选 “Flare ignition: qubodup — CC0”

## blast.wav

内燃机 `blast`（击破爆发）的抢占音效。

- 原始文件：`tmp/fire-sfx/2026-10-27/fire-fuel-explosion-01.wav`
- 处理：仅转 PCM s16le，未变速
- 来源：Mixkit — *Fuel explosion* — https://mixkit.co/free-sound-effects/download/1705/
- 许可：Mixkit Sound Effects Free License，见同目录 `LICENSE-MIXKIT.txt`
- 限制：可用于最终作品；不得把音效本身单独作为素材库/模板再分发

## gear.wav

内燃机每次升档（档位 1→2 … 5→6）的 mode 2（免疫）音效。

- 原始文件：`tmp/gear-sfx/2026-10-27/gear-shift-tank-01.wav`
- 处理：仅转 PCM s16le，未变速（1.287007s）
- 来源：Mixkit — *Metal tank gear shift* — https://mixkit.co/free-sound-effects/download/2757/
- 许可：Mixkit Sound Effects Free License，见同目录 `LICENSE-MIXKIT.txt`
- 限制：可用于最终作品；不得把音效本身单独作为素材库/模板再分发

## cruise.wav

内燃机到 6 档并实际达到最高巡航速度（300 的 98%）时播放一次；循环周期内只响一次，虚弱结束回 1 档后可再次触发。

- 原始文件：`tmp/fire-sfx/2026-10-27/fire-burn-burst-01.wav`
- 处理：仅转 PCM s16le，未变速（3.510000s）
- 来源：Mixkit — *Big fire burning* — https://mixkit.co/free-sound-effects/download/1335/
- 许可：Mixkit Sound Effects Free License，见同目录 `LICENSE-MIXKIT.txt`
- 限制：可用于最终作品；不得把音效本身单独作为素材库/模板再分发

## 音量

`attack.wav`、`blast.wav`、`gear.wav`、`cruise.wav` 已统一到 -15.7 LUFS
（以地慧星斩击 `character/glitch/fx/iai.mp3` 为基准），真实峰值不超过 -1.0 dBFS。
完整测量表见 `docs/audio-loudness.md`。
