# 时计音效

- `moment.wav`：时计落下新时刻时的铃音。源文件 `alarm-ring-07.wav`（SFXMint，双铃闹钟短响，2.0s），CC0 1.0。
- 处理：44.1kHz / 16-bit PCM 立体声；响度归一到 -15.7 LUFS，真峰值 -4.2 dBFS。
- 出处：https://sfxmint.com/dl/office-alarm-clock-14.wav
- 许可：CC0 1.0，见同目录 `LICENSE-CC0-1.0.txt`。

代码里这个声音走 `mode 2`（禁止抢占），不参与抢占，也不会被低优先级抢占音截断。

- `revive.wav`：时计从时刻回拨复活时的滴答声。源文件 `clock-tick-07.wav`（OpenGameArt bart `ticks.zip`，0.711s，四连滴答），CC0 1.0。
- 处理：44.1kHz / 16-bit PCM 立体声；短音按峰值归一到 -1.0 dBFS（积分响度约 -19.6 LUFS）。
- 出处：https://opengameart.org/content/clock-ticks
- 许可：CC0 1.0，见同目录 `LICENSE-CC0-1.0.txt`。
- 同样走 `mode 2`，只在 `rewind` 复活时触发一次。
