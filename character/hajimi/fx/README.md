# 反重力猫咪音效

## 触发与声音

- 开场：本体第一份感知发出 `hajimi-meow`，播放 `meow.wav`，后续感知帧不重复。采用约 0.46 秒的单声猫叫，保留原音高，中频略突出，不叠声、不加回响。
- 普通星弹：实际发射时的 `hajimi-fire` 播放 `twinkle.wav`，一声轻短的高音钟铃闪光。
- 每 20 秒开启三秒重力脉冲：`gravity-field` 同时播放猫叫与 `starlight.wav`。星光音由原始魔法钟铃错开叠成左右声道，加上升音高与短回响，配合双环和星屑展开。
- 弹体扭转、敌弹转化、己方弹捕获：`gravity-twist` / `gravity-convert` / `gravity-capture` 共用更轻的短闪光，播放增益分别为 0.25 / 0.375 / 0.325。
- 场结束时释放环绕弹：`gravity-release` 播放完整星光音，增益 0.48；同一批弹只响一次。

`shot.js` 沿用角色私产声音约定：从当前脚本 `data-pack` 读取 WAV，经 `arena.voice` 仲裁；重触发统一 `stop → reset → play`。猫叫与完整星光使用 IMMUNE，短闪光使用 NORMAL / 205。播放音量分别为 1.0、0.6、0.5，密集事件按比赛时间分别限制为每 750ms、350ms、120ms 一次；声道池为 2、2、3。新对局时间归零时不继承上一场冷却。

素材均为一次性短音，不使用循环、定时器续播或运行时变调。所有音高、分层与回响已烘焙进文件，普通对局与导出使用同一份资源。

## 来源与许可

- `meow.wav`：IgnasD 的 **Meow**，原始文件 `Meow.ogg`，**CC0 1.0**，免费可商用，无强制署名。以短促、清楚的一声喵为方向，未使用泰拉瑞亚的游戏音频。
  - https://opengameart.org/content/meow
- `twinkle.wav`、`starlight.wav`：**The Berklee College of Music** 的 **Shimmer glitter magic**，由 qubodup 发布，原始文件 `shimmer_1.flac`，**CC BY 3.0**，允许免费商用和改编，需要保留署名、来源和许可链接。署名与改编说明已写入随包的 `ATTRIBUTION.md`。
  - https://opengameart.org/content/shimmer-glitter-magic
  - https://creativecommons.org/licenses/by/3.0/

2026-10-30 已核验。原始发布页的纯文本记录保存在 `license/`，CC0 全文为 `LICENSE-CC0-1.0.txt`。逐文件原始/成品 SHA-256、精确处理参数、时长、LUFS 和真峰值见 `audio-sources.json`。

处理后格式统一为 44.1kHz / 16-bit / stereo PCM WAV，加入淡入淡出、6.5–7.2kHz 低通和增益控制。猫叫通过轻度中频提升和动态压缩突出主体，为 -13.5 LUFS、真峰值 -5.9dBFS；计入播放音量后，积分响度比旧版约提高 7.4dB。短闪光按真峰值控制到 -8dBFS，完整星光为 -20 LUFS。
