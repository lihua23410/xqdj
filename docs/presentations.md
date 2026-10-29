# 开场演出接入

开场演出使用与战斗模拟独立的协议，普通观战和视频导出共用同一个资源包播放器。目前只支持正式比赛之前的演出，可以连续提交多段；不支持局中插播。

## 服务端提交

演出生产者负责准备数据，开场编排在 `Match.Start()` 之前调用：

```go
id, err := match.QueueOpening(replay.Presentation{
    Pack: "资源包名称",
    Type: "banner",
    Duration: 1.5, // 秒，有限正数
    Audio: false,
    Data: json.RawMessage(`{"message":"准备开场"}`),
})
```

资源包必须已经通过 `unit.NewPack` 登记；`Type` 在该包内唯一。`Data` 可为任意有效 JSON，由该包的播放器解释。提交时复制数据并生成跨场不复用的 ID。局中提交返回错误。战斗机/场地包通过 `unit.Presentation`（同一协议的别名）返回描述交给编排层，只依赖现有 `unit` 协议，无需依赖 `sim` 或视频导出包。

每场按提交顺序播放。最后一段完成时页面发送 `presentation-end` 和该段 ID；服务端忽略前段或上一场的迟到回执。没有浏览器回执时，按总时长加等待余量释放比赛。录像独立保存整组描述和真局初态，直播结束和下一局不会改写旧导出任务。

## 包内播放器

把脚本放在该包已经 embed 的 `fx/` 下，页面加载包资源时注册：

```js
arena.registerPresentation("banner", {
  mount(ctx, data) {
    const el = document.createElement("div");
    el.textContent = data.message;
    ctx.root.appendChild(el);
    return el;
  },
  render(el, elapsed, ctx) {
    el.style.opacity = String(Math.min(1, elapsed / 0.2));
  },
  destroy(el, ctx) {
    el.remove();
  },
});
```

注册时根据当前脚本绑定资源包，不同包可使用相同的类型名称。必须实现三个生命周期方法；缺少或重复注册会明确报错，导出遇到未知播放器时失败。

`mount` 返回播放器私有状态；`render` 每次根据秒数 `elapsed` 和数据绘制，不能依赖调用次数或真实时间。导出需要预览首帧并重新挂载正式演出，播放器必须允许反复创建和销毁。

`ctx` 提供：

- `root`：本段专属容器，离段后自动移除。
- `id`、`duration`、`elapsed`、`mode`（`live` 或 `export`）、`preview`。
- `scale/cx/cy`、`screenPos` 和战斗机外观相关绘制辅助函数。
- `animate(el, keyframes, options)`：暂停并按段内进度定位的原生动画，销毁时自动取消。
- `listen(target, event, fn, options)`：离段后自动解绑的事件监听。
- `own(dispose)`：登记其他资源的清理函数，按逆序回收。
- `playAudio(src, options)`：遵守描述中的 `Audio` 开关，预览静音，离段自动停止。

剧情进度只由 `elapsed` 决定。不要直接用 `Date.now`、`performance.now`、`setTimeout`、`requestAnimationFrame` 或自行创建音频推进剧情；外部资源需要通过上述入口或 `own` 管理。播放器自行创建的皮肤挂件、GSAP 时间线等，应在 `destroy` 中清理；即使播放器抛错，调度器仍会回收已登记资源并移除容器。

## 非因果与验证

`character/acausal/presentation.go` 负责倒带描述和时长，`fx/rewind.js` 负责轨迹选择及绘制，相关样式也归该包。通用录像、调度器和导出器不再解释死亡时间或倒带倍率。

常规验证：`go test ./...` 和 `node --test internal/web/*.test.cjs`。

真实浏览器/编码验证：设置 `XQDJ_EXPORT_SMOKE=1` 后运行 `go test ./internal/export -run TestRenderSmoke -v`，需要 Chromium 与 ffmpeg。可用 `XQDJ_EXPORT_ARTIFACTS` 指定截图和短片目录。测试包含普通局、非因果单挑/团队赛、独立文本演出及文本接倒带序列；测试播放器只在测试资源包中登记，不进入正式二进制。
