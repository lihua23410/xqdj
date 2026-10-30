# 内燃机 FX 资源

## 球皮与火焰

`engine.js` / `engine.css` 绘制四向活塞、散热鳍片和中央飞轮。1→6 档逐渐从钢灰金属升温到橙红炽热，泵动频率从 1.48Hz 提升到 3.38Hz；球外细槽继续显示热力条。虚弱时机构停转、焰环熄灭，机壳保留暗淡余热。

两层交错火焰环和最多 20 枚余烬复用固定节点，扩散半径跟随当前视野。机械与火焰都由比赛时间推进，暂停时保持同一帧，导出时沿用同一逻辑；卸载球皮会移除全部私有节点与动画状态。

击破爆发由白热核心、火焰膨胀、双层冲击波、四缸定向喷焰、9 枚金属碎片、17 枚火星和 8 团余烟组成；一次创建 43 个短寿命特效节点，最迟约 1.33 秒自动清理。特效半径随击破范围缩放，不改变伤害、时机或音效。

## attack.wav

内燃机普通攻击的抢占音效。

- 原始文件：`tmp/fire-sfx/2026-10-27/fire-flare-ignition-01.wav`
- 处理：`ffmpeg -filter:a atempo=2.0`（2 倍速，保留音高），44.1kHz / 16-bit / stereo
- 后续柔化：3.2kHz 附近削减 5 dB、5kHz 高频搁架削减 5 dB、7kHz 低通，并加入轻度瞬态压缩、5ms 淡入和 30ms 淡出；保持原音高和 0.580499 秒时长。
- 柔化滤镜：`equalizer=f=3200:t=q:w=0.8:g=-5,highshelf=f=5000:g=-5,lowpass=f=7000:p=2,acompressor=threshold=0.25:ratio=2.5:attack=3:release=60:makeup=1,afade=t=in:st=0:d=0.005,afade=t=out:st=0.550499:d=0.03`。输入为已统一响度的旧版 `attack.wav`，不再重新归一化增益。
- 来源：OpenGameArt — *Flare ignition* — qubodup
  https://opengameart.org/content/flare-ignition
- 许可：CC0 1.0 Universal，全文见同目录 `LICENSE-CC0-1.0.txt`
- 署名：不强制；可选 “Flare ignition: qubodup — CC0”

## blast.wav

内燃机 `blast`（击破爆发）的抢占音效。

- 原始文件：`tmp/fire-sfx/2026-10-27/fire-fuel-explosion-01.wav`
- 处理：仅转 PCM s16le，未变速
- 来源：Mixkit — *Fuel explosion* — https://mixkit.co/free-sound-effects/download/1705/
- 许可：Mixkit Sound Effects Free License，见同目录 `LICENSE-MIXKIT.txt`
- 限制：可用于最终作品；不得把音效本身单独作为素材库/模板再分发

## gear.wav

内燃机每次升档（档位 1→2 … 5→6）的 mode 2（免疫）音效。

- 原始文件：`tmp/gear-sfx/2026-10-27/gear-shift-tank-01.wav`
- 处理：仅转 PCM s16le，未变速（1.287007s）
- 来源：Mixkit — *Metal tank gear shift* — https://mixkit.co/free-sound-effects/download/2757/
- 许可：Mixkit Sound Effects Free License，见同目录 `LICENSE-MIXKIT.txt`
- 限制：可用于最终作品；不得把音效本身单独作为素材库/模板再分发

## cruise.wav

内燃机到 6 档并实际达到最高巡航速度（300 的 98%）时播放一次；循环周期内只响一次，虚弱结束回 1 档后可再次触发。

- 原始文件：`tmp/fire-sfx/2026-10-27/fire-burn-burst-01.wav`
- 处理：仅转 PCM s16le，未变速（3.510000s）
- 来源：Mixkit — *Big fire burning* — https://mixkit.co/free-sound-effects/download/1335/
- 许可：Mixkit Sound Effects Free License，见同目录 `LICENSE-MIXKIT.txt`
- 限制：可用于最终作品；不得把音效本身单独作为素材库/模板再分发

## 音量

`blast.wav`、`gear.wav`、`cruise.wav` 已统一到 -15.7 LUFS
（以地慧星斩击 `character/glitch/fx/iai.mp3` 为基准），真实峰值不超过 -1.0 dBFS。
完整测量表见 `docs/audio-loudness.md`。

`attack.wav` 柔化后测得 -19.4 LUFS、真峰值 -6.7 dBFS。普通攻击播放时继续使用 `volume: 0.5`（额外降低约 6 dB），保留点火声的主体、减轻高频刺耳感；其余内燃机音效保持原音量。
