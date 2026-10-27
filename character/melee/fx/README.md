# 原型机_近战 FX 资源

## dash.wav

原型机_近战触发突进（`dash` FX）时的加速音效；mode 2（免疫/禁止抢占），priority 0。

- 原始文件：`tmp/scifi-accel-sfx/2026-10-28/scifi-accel-06.wav`
- 处理：`volume=2.0dB`（+2.0dB，受真峰值限制），44.1kHz / 16-bit / stereo，1.208322s
- 响度：-15.9 LUFS，真峰值 -1.0 dBFS（基准 `character/glitch/fx/iai.mp3`，目标 -15.7 LUFS）
- 来源：Mixkit — *Fast sci-fi transition sweep* — https://mixkit.co/free-sound-effects/download/3114/
- 许可：Mixkit Sound Effects Free License，见同目录 `LICENSE-MIXKIT.txt`
- 限制：可用于最终作品；不得把音效本身单独作为素材库/模板再分发

## attack.wav

原型机_近战弧命中敌人时的攻击音效；mode 1（抢占），priority 199（比受击音效的 200 小 1）。

- 原始文件：`tmp/scifi-accel-sfx/2026-10-28/scifi-accel-15.wav`
- 处理：`volume=0.4dB`（按真峰值限制到 -1.0 dBFS），44.1kHz / 16-bit / stereo，0.341723s
- 响度：-70.0 LUFS，真峰值 -1.0 dBFS；这是 0.34 秒的短脉冲，积分响度不具参考意义，按峰值归一
- 来源：OpenGameArt — *Boost / Launch / Thruster Sound Effect* — EZduzziteh
  https://opengameart.org/content/boost-or-launch-or-thruster-sound-effect
- 许可：CC0 1.0 Universal，全文见同目录 `LICENSE-CC0-1.0.txt`
- 署名：不强制；可选 “Boost SFX: EZduzziteh — CC0”

## shot.js

`dash` 是突进事件：敌人进入视野并触发 `SetVelocity` 时，`原型机_近战.go` 会发
`FX{Name: "dash"}`。`shot.js` 在该事件里播放 `dash.wav`，并保留原有的
`fx-shock` / `fx-ring` / `burst` 视觉。

`attack` 是近战弧命中事件：`近战弧.Handle` 在确认伤害后发 `FX{Name: "attack"}`，
`shot.js` 播放 `attack.wav`。
