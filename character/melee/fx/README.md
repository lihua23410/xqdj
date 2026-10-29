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
`FX{Name: "dash"}`。`shot.js` 在该事件里播放 `dash.wav`，并绘制朝向对应的青色推进焰和少量粒子。

`attack` 是近战弧命中事件：`近战弧.Handle` 在确认伤害后发 `FX{Name: "attack"}`，
`shot.js` 播放 `attack.wav`。

## melee.js / melee.css

本体使用深青装甲、肩部护板、散热格栅和冰青色光学核心；随实际移动方向转向，加速时增强尾部推进器。

视野圈由私有 `guide` 绘制分段刻度、方向标和从中心向外扩散的三道水波纹，不使用旋转雷达扫线。外沿对应快照的实际 `vision`，范围数值同步显示；目标选择复用页面的 `seekTarget`，保留可瞄准优先度和视线遮挡判断，锁定时显示冰青色锁定框。

保持 `VisionRing` 元数据，复用原生 `ring-<id>` 容器，避免重复画圈。常驻水波纹按模拟时间计算，暂停冻结、导出一致；引导层沿用公共清理生命周期，皮肤卸载时回收装甲和视野圈。整套外观仅使用青色系，战斗数值、索敌和碰撞逻辑未修改。
