# 原型机_远程 FX 资源

## shot.wav

原型机_远程发射子弹（`shot` FX）时的音效；mode 2（免疫/禁止抢占），priority 0。

- 原始文件：`tmp/scifi-accel-sfx/2026-10-28/scifi-accel-10.wav`
- 处理：`atempo=1.635691`（整段 2.453537s 时间压缩到 1.5s，保留音高）+ `atrim=0:1.5` + 尾部 50ms 淡出 + `volume=-0.3dB`，44.1kHz / 16-bit / stereo，1.500000s
- 响度：-17.3 LUFS，真峰值 -1.0 dBFS（基准 `character/glitch/fx/iai.mp3`，目标 -15.7 LUFS）
- 来源：Mixkit — *Sci-fi plasma power-up* — https://mixkit.co/free-sound-effects/download/1679/
- 许可：Mixkit Sound Effects Free License，见同目录 `LICENSE-MIXKIT.txt`
- 限制：可用于最终作品；不得把音效本身单独作为素材库/模板再分发

## ranged.js / ranged.css

本体使用简洁的哑光橙色机甲炮台：纯色装甲块面、简化履带底座、独立瞄准的炮塔和长炮管。
底盘跟随移动方向，炮塔按现有 `seekTarget` 的优先度、视野与遮挡规则指向敌人；
开火后向后移动时，炮管仍朝向射击目标。全场视野不绘制大范围视野圈。

真实 `shot` FX 驱动炮管后坐与复进、短促炮口焰、侧面散热条和 1.6 秒装填细环。
装填完成后细环隐藏，目标只显示低亮度四角准星；装甲采用纯色，减少高光与细碎装饰。
开火的前 0.16 秒保持真实出膛方向；全部演出使用比赛时间，暂停时冻结，导出可复现。
射击历史保存在本机的私有导引节点上，随单位移除清理；时间回退时重置。
目标不可瞄准、离开视野或被遮挡时撤掉准星，不改变索敌、射击、子弹及碰撞逻辑。

## shot.js

`原型机_远程.go` 每次实际开火时发 `FX{Name: "shot"}`；`shot.js` 播放
`shot.wav`。炮台皮肤接管开火视觉，炮口焰从炮管端口发出。

音效 1.5 秒、开火间隔 1.6 秒，留 0.1 秒余量；下一次开火仍会主动 `stop/reset/play`。
