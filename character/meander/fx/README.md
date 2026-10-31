# 迷走音效

攻击采用细碎的风铃颗粒音，换环采用低柔、空灵的气流过渡音，无循环背景声。

- `attack.wav`：0.32 秒的柔和水晶碎响；已降调、通过两级 3kHz 低通柔化高频，并加入 18ms 淡入与轻压缩，减弱尖锐起音。亮盘实际发出伤害时触发，同拍多个目标只发一次。使用 **mode 1 / PREEMPT、priority 199**；密集事件按对局时间限频 80ms。
- `switch.wav`：0.82 秒的低柔气流与短余韵，已降调约 4.3 个半音，并用 1.8kHz 低通压低高频；实际进入另一条环时触发，包括离轨纠正换环。使用 **mode 2 / IMMUNE、priority 0**。开局上环、交点继续原环、撞墙反绕、两秒锁定期间均不响换环音。

两段声音均在本包 `audio.js` 通过 `arena.voice` 播放，重触发走 `stop → reset → play`，音量为 1。攻击用标准 `attack` FX 按快照到达即时播放，换环用 `meander-switch` FX。两种 FX 的 `Amount` 均携带发生时的对局时间，用于实时观战与导出一致的限频；画面阶段不会重复播放即时攻击音。

## 来源与许可

攻击音改编自 **Magic Wand Glitter**，作者 **qubodup / Iwan Gabovitch**。

- 作者发布页：https://freesound.org/people/qubodup/sounds/211624/
- 许可：**CC0 1.0 Universal**，https://creativecommons.org/publicdomain/zero/1.0/
- 2026-10-31 核验作者页面；页面明确声明自 2026-06-06 起为 CC0。可免费商用、修改、随游戏分发，无强制署名。
- 原合成所用的两段录音也均为 CC0：`chimes bar`（urupin）、`Instrument_chimes_quick_soft.aif`（vrodge）。
- 下载源为作者公开的 HQ MP3 预览：https://cdn.freesound.org/previews/211/211624_71257-hq.mp3 。原始上传格式是 FLAC，本项目没有使用需要登录的原文件下载。
- 可选署名：`Magic Wand Glitter by qubodup, with CC0 chimes by urupin and vrodge — CC0 1.0`。

换环音改编自 **digital_whoosh_soft.wav**，作者 **ludvique**。

- 作者发布页：https://freesound.org/people/ludvique/sounds/71852/
- 许可：**CC0 1.0 Universal**，https://creativecommons.org/publicdomain/zero/1.0/
- 2026-10-31 核验作者页面，可免费商用、修改、随游戏分发，无强制署名。
- 下载源为作者公开的 HQ MP3 预览：https://cdn.freesound.org/previews/71/71852_1062668-hq.mp3 ，原始上传格式是 WAV。
- 可选署名：`digital_whoosh_soft.wav by ludvique — CC0 1.0`。

两段均经过剪辑、均衡、降调、低通、动态压缩、淡入淡出、短回声、增益调整和 44.1kHz / 16-bit / stereo PCM WAV 转换。所有处理已烘焙，无运行时变调或 Web Audio 依赖。完整滤镜、来源与成品 SHA-256、时长、响度见 `audio-sources.json`，页面许可记录见 `license/freesound-211624.txt` 与 `license/freesound-71852.txt`，CC0 全文见 `LICENSE-CC0-1.0.txt`。

攻击音过短，积分 LUFS 不具参考意义；柔化后真峰值降至 -6.1dBFS，保留较轻的颗粒感，不再抬回原先的 -1.1dBFS。换环音为 -15.7 LUFS、真峰值 -6.5dBFS。
