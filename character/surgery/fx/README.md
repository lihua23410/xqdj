# 外科手术

本体保留圆球轮廓，仅以帽沿、口罩褶皱和手术服缝线表现外科医生，不画人形躯干或四肢。伤口、针眼、缝线、刀和血珠由本包 SVG / Canvas / CSS 绘制。地面展示读取 `surgery-state`，不在浏览器推进判定；时间取自对局，暂停与录像回看一致。

三段音效为项目原创的确定性带限噪声与金属/液体瞬态合成，无外部录音或采样，按根目录 MIT 许可提供。运行 `python character/surgery/audio/generate_sfx.py` 可重建 44.1kHz 单声道 PCM WAV。短音按约 -8dBFS 峰值控制，不强行提高极短音的积分响度。每类固定三声部并限频，NORMAL / 215 / volume 1；只在真实切割、穿针和崩开事件触发，暂停不重播。

器械轮廓参考 Swann-Morton No.10 短刀片与 B. Braun 弧针黑线，未下载或复制图片。详见 `docs/surgery-design.md`。
