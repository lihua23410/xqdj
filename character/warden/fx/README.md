# 盾士碎盾音效

- `shatter-full.wav`：真正挡伤破盾时，一次短促的金属撞击、碎裂与甲片细响，约 0.28 秒。
- `shatter-weak.wav`：旧盾换新时，更轻、更短的薄金属剥落声，约 0.18 秒。

两段均为本项目原创的确定性合成素材，没有引用外部录音或采样，随项目按根目录 `LICENSE` 的 MIT 许可提供。合成由非谐波金属模态、带限噪声和错开的碎片瞬态组成，源代码在 `../audio/generate_sfx.py`。需要 NumPy 和 ffmpeg，可在仓库根目录运行：

```powershell
python character/warden/audio/generate_sfx.py
```

输出是 44.1 kHz、单声道、16-bit PCM WAV。两段均为极短音，按项目约定以真峰值约 -1 dBFS 控制，不强行拉高无意义的短音积分响度；测量与 SHA-256 见 `audio-sources.json`。

播放跟随现有 `shatter` 展示事件，普通碎片与弱化碎片分别选择对应文件。两种声音都经过 `arena.voice`，明确使用 mode 2（IMMUNE）、priority 0、volume 1.0。不同盾士各用各的声音实例，同一盾士重触发执行 `stop → reset → play`。

同一对局时间、同一盾士、同一种碎盾事件只播放一次，暂停重复渲染不会重播；时间回退会停止旧声音并清空去重。普通观战与导出均使用同一入口，不修改公共声音引擎或战斗规则。
