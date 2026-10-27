# 缺省受击 / 碰撞音效

这两个文件是全局缺省音效，角色自己的受击音（例如老牧师的尖叫）优先，不叠加。

| 文件 | 用途 | 来源 | 许可证 |
| --- | --- | --- | --- |
| `default-hit.wav` | 受击 | Kenney Impact Sounds · `impactPunch_medium_000.ogg` | CC0 1.0 |
| `default-impact.wav` | 碰撞（身体 / 弹体） | Kenney Impact Sounds · `impactGeneric_light_000.ogg` | CC0 1.0 |

- 来源页：https://www.kenney.nl/assets/impact-sounds
- 许可证：CC0 1.0 Universal，全文见 `LICENSE-CC0-1.0.txt`
- 许可证原文：https://creativecommons.org/publicdomain/zero/1.0/legalcode
- 署名：不强制；可选 “Impact SFX: Kenney (www.kenney.nl) — CC0”
- 原始音效包：`tmp/2026-10-27-hit-sfx.zip`（Kenney 部分）

## 版权与许可

`default-hit.wav`、`default-impact.wav` 均来自 Kenney 的 Impact Sounds 包，
以 CC0 1.0 Universal 公共领域贡献发布。CC0 不要求署名；这里保留来源与
可选的致谢：Impact SFX: Kenney (www.kenney.nl) — CC0。

## 音量

这两个缺省音效与角色音效统一到同一响度标准：以 `character/glitch/fx/iai.mp3`
（地慧星斩击）的 -15.7 LUFS 为基准，真实峰值不超过 -1.0 dBFS。
`app.js` 中 `DEFAULT_SFX` 的 `volume` 已设为 `1.0`，不再额外压低。
完整测量表与处理方法见 `docs/audio-loudness.md`。
