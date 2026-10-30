# 回声音效

## 状态与声音

回声有扫描、短暂瞄准、突进三个本体状态；每名敌人各自有独立的扫描层与六秒锁定窗口。星弹和碎片均不出伤，只有本体突进撞击造成伤害。当前保留 10 组私产音效，突进起步与退出锁定均静音。

- 扫描启动：`echo-scan` → `scan.wav`，柔和单音声呐；出生后的第一份感知只响一次。移动时每 1.8 秒实际发射一枚星弹，`echo-fire` → `fire.wav`，短促电子发射声；静止或锁定期间不发射、不空响。
- 星弹爆散：`echo-burst` → `burst.wav`，声场展开音；撞墙或碰撞引爆，配合金色扩散圈。爆散不使用重型爆炸声，避免让人误以为星弹会出伤。
- 扫描 1–9 层：`echo-ping` → `ping.wav`，清晰短音，播放增益随层数略升。最多每 60ms 响一次，使用三个声道轮换，避免多枚碎片同时命中时叠成噪声。
- 真正建立六秒锁定窗口：`echo-found` → `lock.wav`，双音确认。由 `syncLocks` 创建新窗口时发出；同一窗口不刷新、不重复响。第十层的 `echo-ping` 只画白圈，由 `echo-found` 负责锁定声音。
- 初次锁定瞄准：`echo-lock` → `aim.wav`，轻确认声，对应 0.08 秒站定。撞墙后的 0.10 秒重新瞄准由撞墙提示接续，不额外播放满层确认。
- 每次突进：`echo-dash` 保留青色冲刺痕，只播放视觉效果。
- 突进撞中敌人：`echo-ram` → `ram.wav`，带金属质感的短撞击。与当次有效的 6 点伤害同处一个分支，一次突进只响一次。
- 突进撞墙：`echo-wall` → `wall.wav`，较轻的金属收束声，对应回到重新瞄准间隔。
- 最后一个锁定到期或目标消失、回到扫描：不播放声音。
- 碎片前八次反弹：`echo-bounce` → `bounce.wav`，很轻的短音；最多每 90ms 一次。第九次撞墙只播放消散提示。
- 星弹或碎片寿命耗尽、碎片反弹耗尽：`echo-fade` → `fade.wav`，轻微回落音；实体消失后不会再次触发。扫描层自然衰减或未锁定扫描环归零也使用该声音，最多每 150ms 一次。
- 受击、普通身体碰撞继续使用引擎公共音效。静止、暂停和倒下没有持续循环音；不会在撤掉球皮时凭空播放“锁定结束”。

## 播放约定

所有声音均为角色私产，资源从当前脚本的 `data-pack` 下读取，只经过 `arena.voice` 播放；每次重触发均为 `stop → reset → play`，不直接调用裸 `Audio.play()`。

- `ram` 使用 PREEMPT / 130。
- `ping`、`lock`、`aim`、`wall` 使用 IMMUNE，保证扫描与状态切换提示能听见；它们仍遵守全局 FORCE 仲裁。
- `scan`、`fire`、`burst`、`bounce`、`fade` 使用 NORMAL，优先级分别为 210、205、205、220、220。
- 所有素材都是短音，没有循环声或计时器生成的自动音序。密集事件按比赛时间限频，正常对局和导出使用相同时间；新对局时间归零后可立即触发。
- 没有运行时变速或变调。源素材的保音高变速、柔化滤波和增益均已烘焙进 WAV，导出不会丢掉这些处理。

## 素材与许可

作者均为 **Kenney / Kenney Vleugels**。2026-10-30 核验作者本人发布页的 **CC0 1.0** 标记、官网商用说明及包内许可；均可免费商用、修改和随作品分发，无强制署名。可选署名为 `Sound effects by Kenney (kenney.nl), CC0`。

- Digital SFX：`scan` 来自 `tone1.mp3`，`lock` 来自 `twoTone1.mp3`。
  - 作者发布页：https://opengameart.org/content/63-digital-sound-effects-lasers-phasers-space-etc
  - 原包：https://opengameart.org/sites/default/files/Digital_SFX_Set.zip
  - 官网：https://kenney.nl/assets/digital-audio
  - 原包商用许可：`LICENSE-KENNEY-DIGITAL.txt`。
- Sci-Fi Sounds 1.0：`fire` 来自 `laserSmall_003.ogg`，`burst` 来自 `forceField_000.ogg`，`ram` 来自 `impactMetal_002.ogg`，`wall` 来自 `impactMetal_000.ogg`。
  - 作者发布页：https://opengameart.org/content/sci-fi-sounds
  - 原包：https://opengameart.org/sites/default/files/sci-fi_sounds.zip
  - 官网：https://kenney.nl/assets/sci-fi-sounds
  - 原包许可：`LICENSE-KENNEY-SCIFI.txt`。
- Interface Sounds 1.0：`ping` 来自 `pluck_002.ogg`，`aim` 来自 `confirmation_001.ogg`，`bounce` 来自 `pluck_001.ogg`，`fade` 来自 `minimize_001.ogg`。
  - 作者发布页：https://opengameart.org/content/interface-sounds
  - 原包：https://opengameart.org/sites/default/files/kenney_interfaceSounds.zip
  - 官网：https://kenney.nl/assets/interface-sounds
  - 原包许可：`LICENSE-KENNEY-INTERFACE.txt`。

官网商用说明：https://kenney.nl/support 。CC0 全文见 `LICENSE-CC0-1.0.txt`。

处理后的文件均为 44.1kHz / 16-bit / stereo PCM WAV。统一对 80Hz 以下低频与 6.5kHz 以上高频做柔化滤波，加入短淡入淡出；频繁事件额外降低播放音量。逐文件来源、原始与成品 SHA-256、精确滤镜、时长、LUFS 和真峰值记录在 `audio-sources.json`。极短提示音的积分 LUFS 不具参考意义，按真峰值控制；所有文件真峰值不超过 -2.5dBFS。
