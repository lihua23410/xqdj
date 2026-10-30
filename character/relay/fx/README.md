# 中转站 · IGP 路由皮肤与音效

交换机特征直接融入球壳：下半球沿弧面排列嵌入式端口，上半球是成对上联口，两侧开短散热槽，中央保留双向路由符号。随从采用同样的球面布局，端口减到三个。没有独立机箱、矩形前面板或 L3 字牌。球壳、拓扑、瞬时特效和报文使用原有琥珀、铜棕、浅金色系；红绿仅用于小面积状态灯。绿色链路灯随发送变亮，暗红诊断灯罩在上线和接收时亮绿，随实际路由活动响应。HELLO 上线、接收同步与逐跳传播保留 IGP 意象。参考 [OSPF RFC 2328](https://www.rfc-editor.org/rfc/rfc2328.html) 与 [Integrated IS-IS RFC 1195](https://www.rfc-editor.org/rfc/rfc1195.html)。

这次更新只增加演出事件，不改变选下一跳、出弹上限、接球距离、伤害、速度或随从生成间隔。游戏仍按既有的“未接过优先、较久未接优先、再比较距离”选下一跳；没有实现 SPF 或真实协议状态机。

## 表现与触发

- `relay-join`：随从第一次 Sense，上线 HELLO 单环与双音提示；每只随从一次。
- `relay-forward`：成功生成中转弹后，发送指示灯与端口触点短暂变亮，实际下一跳链路亮起，播放轻短数据发送声。`unitId` 是发出者，`amount` 是已选定的目标 ID，`vx/vy` 为当时目标坐标。
- `relay-arrive`：弹进入目标接球范围、被接收后，浅金同步环收拢并播放确认声；没有下一跳也有接收表现。
- `relay-hit`：实际有效伤害时，目标处数据碎片闪开并播放短促冲击声，遵守原来的 0.1 秒伤害冷却。

细暗虚线仅作邻居拓扑背景：每个随从连一个较早节点，节点集变化时才重建。亮琥珀链路严格取实际转发目标，0.7 秒淡出，跨本体的随从不互连。SVG 线条复用，不逐帧累积。收发亮度、01 数字闪动、拖尾位移和虚线相位读比赛时间，暂停与导出保持一致；新局时间回退会清空旧活动。

中转弹直接显示闪动的 `01`，两个数字交替明暗，后方跟随逐渐变小、变淡的二进制数字拖尾；不再使用实体箭头包壳。数字始终正向可读，拖尾按实际飞行方向排列。闪动与拖尾位移读取比赛时间，暂停时保持同一帧。同一节点 0.12 秒内的收发瞬时闪环合并，收发亮度与实际下一跳仍逐事件更新。瞬时特效均在根节点设置有限 CSS 动画，由通用 `spawnFx` 自动回收。

## 音效约定

使用角色私产 `arena.registerShot`、`arena.voice(new Audio(...))`，每次播放执行 `stop → reset → play`。所有处理预先烘焙到 WAV，运行时不变调、不改速，导出可使用同一音源。

- `neighbor.wav`：IMMUNE，音量 0.6，2 个声音实例，最短间隔 0.35 秒。
- `forward.wav` / `ack.wav`：NORMAL，优先级 210，音量 0.38，各 2 个声音实例；共用 0.16 秒最短间隔，密集收发也不会叠满提示音。
- `hit.wav`：NORMAL，优先级 205，音量 0.55，2 个声音实例，最短间隔 0.16 秒。

音频限频不影响收发和路由状态。无持续循环音。短音按真峰值控制，邻居上线提示约 -21 LUFS；详细滤镜、原始文件、SHA-256、输出响度与长度见 [audio-sources.json](audio-sources.json)。

## 免费商用素材

四个音效均由 Kenney (Kenney Vleugels) 创作，原素材 CC0-1.0，可免费商用及修改，无强制署名。2026-10-30 核验。处理为剪裁、保音高加速、低高频削减、淡入淡出和音量调整。

- `neighbor.wav` ← `twoTone2.mp3`：[Digital SFX 原发布页](https://opengameart.org/content/63-digital-sound-effects-lasers-phasers-space-etc)，[Kenney Digital Audio](https://kenney.nl/assets/digital-audio)。
- `forward.wav` ← `pluck_001.ogg`；`ack.wav` ← `confirmation_003.ogg`：[Kenney Interface Sounds](https://kenney.nl/assets/interface-sounds)。
- `hit.wav` ← `impactMetal_004.ogg`：[Kenney Sci-Fi Sounds](https://kenney.nl/assets/sci-fi-sounds)。

保留素材包原始许可 `LICENSE-KENNEY-*.txt` 与 CC0 正文 `LICENSE-CC0-1.0.txt`。
