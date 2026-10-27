# forge.wav

炭翁**自己生成炭**时播放的一次性音效。


- 触发点：`shot.js` 的 `forge(host, added)`，只在 `holder === ownerId` 时调用。
- 源文件：`tmp/fire-sfx/2026-10-27/fire-crackle-short-01.wav`
  - 来源：OpenGameArt / AntumDeluge "Fire Crackling"
  - 许可：CC0 1.0（见 `LICENSE-CC0-1.0.txt`）
- 处理：2.870612s 压缩到 1.000s
  - `atempo=1.913741,atempo=1.5`（保持音高，整体加快）
  - 44.1kHz / 16-bit PCM / 立体声
  - `volume=+12.5dB`，真峰值 -1.0 dBFS，积分 -20.4 LUFS
  - 末尾 60ms 淡出
- 声音模式：`mode 2 (IMMUNE)`，`priority 0`；可叠在其它声音上，自身不会被 mode 1 打断。

# hand.wav

炭翁**把炭交给目标**时播放的一次性音效，与 `hand` 飞行动画同一触发。

- 触发点：`shot.js` 的 `hand` 钩子。
- 源文件：`tmp/throw-sfx/2026-10-28/throw-09.wav`
  - 来源：OpenGameArt / artisticdude "Swishes Sound Pack"（swish-2）
  - 许可：CC0 1.0（见 `LICENSE-CC0-1.0.txt`）
- 处理：0.095828s，44.1kHz / 16-bit PCM / 立体声
  - `volume=-0.9dB`；平均电平 -15.1 dB，最大 -1.0 dBFS
  - 极短音，EBU R128 积分门限会显示 -70 LUFS，不代表无声
- 声音模式：`mode 2 (IMMUNE)`，`priority 0`；可叠在其它声音上，自身不会被 mode 1 打断。

# burn.wav

炭**在目标身上烧掉、造成伤害**时的音效。

- 触发点：`shot.js` 的 `burn()`（`coal` 数量减少且不是 `hand` 交接的那一帧）。
- 源文件：`tmp/fire-sfx/2026-10-27/fire-swoosh-burning-01.wav`
  - 来源：Mixkit "Fire Swoosh Burning"
  - 许可：Mixkit Sound Effects Free License（https://mixkit.co/license/），不是 CC0
- 处理：截取原文件 **2.0s 到结尾**，1.631927s；44.1kHz / 16-bit PCM / 立体声
  - `loudnorm=I=-15.7:TP=-1.0` 后 `volume=+7.8dB` + `alimiter`，积分 -16.0 LUFS、真峰值 -1.0 dBFS
  - 开头 10ms 淡入，末尾 80ms 淡出
- 声音模式：`mode 1 (PREEMPT)`，`priority 199`。

