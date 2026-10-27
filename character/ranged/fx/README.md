# 原型机_远程 FX 资源

## shot.wav

原型机_远程发射子弹（`shot` FX）时的音效；mode 2（免疫/禁止抢占），priority 0。

- 原始文件：`tmp/scifi-accel-sfx/2026-10-28/scifi-accel-10.wav`
- 处理：`atempo=1.635691`（整段 2.453537s 时间压缩到 1.5s，保留音高）+ `atrim=0:1.5` + 尾部 50ms 淡出 + `volume=-0.3dB`，44.1kHz / 16-bit / stereo，1.500000s
- 响度：-17.3 LUFS，真峰值 -1.0 dBFS（基准 `character/glitch/fx/iai.mp3`，目标 -15.7 LUFS）
- 来源：Mixkit — *Sci-fi plasma power-up* — https://mixkit.co/free-sound-effects/download/1679/
- 许可：Mixkit Sound Effects Free License，见同目录 `LICENSE-MIXKIT.txt`
- 限制：可用于最终作品；不得把音效本身单独作为素材库/模板再分发

## shot.js

`原型机_远程.go` 每次实际开火时发 `FX{Name: "shot"}`；`shot.js` 播放
`shot.wav`，并保留原有的 `fx-flash` / `fx-beam` 视觉。

音效 1.5 秒、开火间隔 1.6 秒，留 0.1 秒余量；下一次开火仍会主动 `stop/reset/play`。
