# 音效响度统一

以 `character/glitch/fx/iai.mp3`（地慧星斩击）为基准，把全部游戏音效统一到
**-15.7 LUFS** 的积分响度，真实峰值控制在 **-1.0 dBFS** 以内。

## 方法

1. 用 `ffmpeg -filter_complex ebur128=peak=true` 测量每个文件的积分响度 `I` 与真峰值 `TP`。
2. 目标增益 `gain = -15.7 - I`。
3. 受真峰值限制：`gain = min(gain, -1.0 - TP)`，避免削波。
4. WAV 用 PCM s16le 重写；MP3 用 libmp3lame 192k 重写。
5. 所有播放音量保持 1.0；缺省受击/碰撞原先的 0.32 / 0.16 已取消。

## 结果

| 文件 | 用途 | I (LUFS) | TP (dBFS) |
| --- | --- | ---: | ---: |
| `character/glitch/fx/iai.mp3` | 基准：地慧星斩击 | -15.7 | 0.0 |
| `internal/web/sfx/default-hit.wav` | 缺省受击 | -16.5 | -1.0 |
| `internal/web/sfx/default-impact.wav` | 缺省碰撞 | -70.0 | -1.0 |
| `character/pastor/fx/scream.mp3` | 老牧师受击 | -16.0 | -9.7 |
| `character/pastor/fx/verse0.mp3` | 老牧师经文 0 | -16.0 | -12.5 |
| `character/pastor/fx/verse1.mp3` | 老牧师经文 1 | -15.9 | -11.2 |
| `character/pastor/fx/verse2.mp3` | 老牧师经文 2 | -15.9 | -8.4 |
| `character/pastor/fx/verse3.mp3` | 老牧师经文 3 | -15.9 | -12.1 |
| `character/wolf/fx/animals.mp3` | 狼人嚎叫 | -16.0 | -6.8 |
| `character/engine/fx/attack.wav` | 内燃机普通攻击 | -15.7 | -2.6 |
| `character/engine/fx/blast.wav` | 内燃机击破爆发 | -15.7 | -3.4 |
| `character/engine/fx/gear.wav` | 内燃机升档 | -16.0 | -1.0 |
| `character/engine/fx/cruise.wav` | 内燃机最高档巡航 | -15.7 | -2.3 |

`default-impact.wav` 只有 0.16 秒，积分响度对这类极短点击没有意义；它已按真峰值
限制到 -1.0 dBFS，听感上是一次短促碰撞。`iai.mp3` 作为基准未改动。

原始文件备份在 `tmp/audio-volume-backup/`（已被 `.gitignore` 的 `/tmp/` 覆盖）。
