# 音效响度统一

以 `character/glitch/fx/iai.mp3`（地慧星斩击）为基准，把全部游戏音效统一到
**-15.7 LUFS** 的积分响度，真实峰值控制在 **-1.0 dBFS** 以内。

## 方法

1. 用 `ffmpeg -filter_complex ebur128=peak=true` 测量每个文件的积分响度 `I` 与真峰值 `TP`。
2. 目标增益 `gain = -15.7 - I`。
3. 受真峰值限制：`gain = min(gain, -1.0 - TP)`，避免削波。
4. WAV 用 PCM s16le 重写；MP3 用 libmp3lame 192k 重写。
5. 统一时播放音量为 1.0；内燃机普攻现单独设为 0.5。缺省受击/碰撞原先的 0.32 / 0.16 已取消。

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
| `character/engine/fx/attack.wav` | 内燃机普通攻击（柔化后） | -19.4 | -6.7 |
| `character/engine/fx/blast.wav` | 内燃机击破爆发 | -15.7 | -3.4 |
| `character/engine/fx/gear.wav` | 内燃机升档 | -16.0 | -1.0 |
| `character/engine/fx/cruise.wav` | 内燃机最高档巡航 | -15.7 | -2.3 |
| `character/melee/fx/dash.wav` | 原型机_近战突进加速 | -15.9 | -1.0 |
| `character/melee/fx/attack.wav` | 原型机_近战攻击 | -70.0 | -1.0 |
| `character/ranged/fx/shot.wav` | 原型机_远程发射 | -17.3 | -1.0 |
| `character/tokei/fx/moment.wav` | 时计留下时刻 | -15.7 | -4.2 |
| `character/tokei/fx/revive.wav` | 时计回拨复活 | -19.6 | -1.0 |

`default-impact.wav` 只有 0.16 秒，积分响度对这类极短点击没有意义；它已按真峰值
限制到 -1.0 dBFS，听感上是一次短促碰撞。`character/melee/fx/attack.wav` 同样只有
0.34 秒，按峰值归一到 -1.0 dBFS。`iai.mp3` 作为基准未改动。

内燃机普通攻击在统一响度后单独削减了高频并压柔瞬态，播放音量为 0.5；这项例外不再拉回 -15.7 LUFS。滤镜见 `character/engine/fx/README.md`。

原始文件备份在 `tmp/audio-volume-backup/`（已被 `.gitignore` 的 `/tmp/` 覆盖）。

回声的声呐提示与状态提示使用独立的分级音量：密集扫描、反弹、衰减音更轻，锁定和有效撞击更清楚；极短音按真峰值控制，不强行拉到 -15.7 LUFS。逐文件测量与处理参数见 `character/echo/fx/audio-sources.json`，播放约定与 CC0 来源见同目录 `README.md`。

反重力猫咪的单声猫叫为 -13.5 LUFS、播放音量 1.0，完整星光为 -20 LUFS，短闪光按 -8dBFS 真峰值控制；各事件统一限频。详细测量见 `character/hajimi/fx/audio-sources.json`，猫叫的 CC0 来源和星光的 CC BY 3.0 署名见同目录 `ATTRIBUTION.md`。

中转站的邻居上线音约 -21 LUFS，收发短音按 -11 / -10 dBTP、命中按 -7 dBTP 控制；收发共用 0.16 秒限频，避免拓扑增大后声音过密。详细测量和 CC0 素材来源见 `character/relay/fx/audio-sources.json` 与同目录 `README.md`。
