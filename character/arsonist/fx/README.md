# 纵火者音效

| 文件 | 触发与声音 | 时长 | I (LUFS) | TP (dBFS) |
| --- | --- | ---: | ---: | ---: |
| `oil-break.wav` | 油瓶碎裂：玻璃短碎响、低沉黏稠泼油和几滴落液 | 0.62s | -18.2 | -1.0 |
| `fire-break.wav` | 火瓶碎裂：玻璃碎响与短促点火 | 0.85s | -17.2 | -1.0 |
| `crackle-a.wav` | 实际火面/身上燃烧的轻微噼啪，片段 A | 1.15s | -23.5 | -13.0 |
| `crackle-b.wav` | 同上，交替播放的片段 B | 1.15s | -23.5 | -13.5 |

玻璃与黏稠液体部分为本项目原创的确定性模态、带限噪声与液滴合成，按根目录 MIT 许可提供，没有引用外部玻璃/液体录音。火焰部分复用仓库已有、标注 CC0 的原始录音，原件保存在 `../audio/source/`，不依赖被忽略的 `tmp/` 目录：

- `fire-catch.wav`：themightyglider，**Catching fire**，[来源](https://opengameart.org/content/catching-fire)，CC0；缓存原件由 `flame.ogg` 转成 WAV。
- `fire-crackle.wav`：AntumDeluge，**Fire Crackling**，[来源](https://opengameart.org/content/fire-crackling)，CC0。

CC0 完整文本见 `LICENSE-CC0-1.0.txt`。CC0 原件可修改、再分发，无强制署名；上述作者记录保留。最终文件的来源、滤镜、响度与 SHA-256 见 `audio-sources.json`。

所有文件为 44.1kHz 单声道 PCM16 WAV，已经低通柔化高频、压缩瞬态、处理首尾淡入淡出并测量真峰值。碎瓶音按项目方法在 -15.7 LUFS 目标与 -1 dBTP 上限间取较小增益，保留玻璃瞬态，未强拉满积分响度；背景火声刻意以 -23.5 LUFS 制作，避免持续燃烧盖住战斗提示。播放音量均为 1.0。

`audio.js` 使用 `arena.voice`，mode 0（NORMAL），碎瓶 priority 210、火声 priority 230，不抢占重要声音。固定 6 个 voice 复用，重触发按 `stop → reset → play`。油瓶、火瓶分别限频 0.10/0.15 秒，并按瓶 ID 去重；重复渲染与暂停不会重播同一次碎瓶。燃烧只在真实油火或身上火存在时每隔至少 1 秒请求一个短片段，全场共用限频并交替两段录音，不按油格、单位或纵火者个数叠加，也没有会遗留的永久循环。时间回拨会停止旧声音并重置去重与限频，普通观战和导出共用同一路径。

用 NumPy 和 ffmpeg 可重建全部声音与测量清单：

```bash
python3 character/arsonist/audio/generate_sfx.py
node character/arsonist/arson_audio_test.cjs
```
