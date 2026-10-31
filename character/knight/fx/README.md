# 小骑士音效

`blink.wav` 是约 0.34 秒的短促冲锋破空声，绑定实际瞬移时的 `blink` FX：每次瞬移只响一次，0.5 秒连冲也会逐次触发。落点的 `dash` 只画特效，不再叠一声；冷却、无索敌目标、火环蓄满与普通巡航均不空响。

播放参数为 **mode 2 / IMMUNE、priority 0、volume 1**，使用三个声部轮换，经 `arena.voice` 仲裁。每次重触发均为 `stop → reset → play`，不会截掉其他 mode 0/1 的声音，也不会被它们截掉；仍遵守全局 mode 3。声音通过正常 FX 播放路径进入视频导出，不使用 Web Audio、循环音或运行时变调。

## 来源与许可

- 标题：**Swosh / Whoosh / Air Cut**
- 作者：**qubodup**
- 作者发布页：https://freesound.org/people/qubodup/sounds/60030/
- 许可：**CC0 1.0 Universal**，https://creativecommons.org/publicdomain/zero/1.0/
- 2026-10-31 核验作者页面的 CC0 标记，可免费商用、修改并随游戏分发，无强制署名。
- 来源是作者用竹棍录制的破空声。下载使用公开 HQ MP3 预览：https://cdn.freesound.org/previews/60/60030_71257-hq.mp3 ，原始上传格式为 FLAC。
- 可选署名：`Swosh / Whoosh / Air Cut by qubodup — CC0 1.0`。

素材经过剪辑、保音高加速、110Hz 高通、4.2kHz 低通、轻压缩、淡入淡出和增益调整，转换为 44.1kHz / 16-bit / stereo PCM WAV。短音积分 LUFS 不具参考意义，按真峰值 -3.1dBFS 控制，减少尖锐高频并保留冲锋瞬态。

逐项处理参数、时长、响度、来源与成品 SHA-256 见 `audio-sources.json`，页面许可记录见 `license/freesound-60030.txt`，CC0 全文见 `LICENSE-CC0-1.0.txt`。
