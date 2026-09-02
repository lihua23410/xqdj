# 小球对战

六边形场地里的 1v1 观战模拟器。两边各选一只小球，引擎按 60Hz 扫掠碰撞推进，页面只负责显示。

当前是单机本地服：Go 进程里跑对局，浏览器经 WebSocket 收看快照。没有玩家操作、没有网络同步。

## 跑起来

需要 Go 1.22+。

```bash
go run .
```

打开 [http://127.0.0.1:8080](http://127.0.0.1:8080)。拉不下来依赖时：

```bash
GOPROXY=https://goproxy.cn,direct go run .
```

前端文件用 `go:embed` 打进二进制。改了 `internal/web/` 下的 html/css/js 必须重启进程，浏览器再强制刷新。端口被旧进程占住时：

```bash
fuser -k 8080/tcp
go run .
```

测试：

```bash
go test ./...
```

## 目录

```
character/          角色实现（每只球一个文件，init 里注册）
internal/unit/      Actor / Cmd / Sense / 角色类型，sim 与 character 的唯一协议
internal/sim/       物理、对局状态机、CCD、墙
internal/web/       选人页 + 战场（embed）
main.go             HTTP :8080 和 /ws
PLAN.md             尚未做的混战 / 分队
```

`main.go` 用 `_ "xqdj/character"` 拉起所有 `init()`。生产代码里 **`internal/sim` 不要 import `character`**，只走 `internal/unit`。测试里可以 import，用来点名某只球。

## 现有角色

| Kind | 巡航 | 视野 | 行为概要 |
| --- | --- | --- | --- |
| `原型机_近战` | 200 | 185 | 敌方进入视野边沿时朝对方最多转 15°，沿当前朝向 +155；碰撞伤 7，0.1s CD |
| `原型机_远程` | 140 | 9999 | 每 2s 射一发；子弹速 160、伤 8、撞墙 3 次后消失；开火后自身速度反向 |
| `分身者` | 160 | 0 | 本体碰撞伤 0；3 个分身各伤 5；本体挨打且未死时与随机分身换位置 |
| `筑墙者` | 150 | 9999 | 第 3 秒起每 3s 在与敌方中点砌一段长 150 的胶囊墙，存活 7s，伤 3 |
| `无下限术士` | 155 | 9999 | 开局裂成红蓝两个半圆；红吸引、蓝推开。红蓝相撞时朝敌人打一发紫弹（体积约 5.6 倍、速 320、伤 14），穿透敌人可反复结算，穿出六边形后飞出屏幕才消失 |
| `小骑士` | 175 | 9999 | 开局 75 血；碰撞伤 10。每 7s 瞬移到敌人身边（不出六边形）再朝对方冲撞。放完有 50% 概率 0.5s 后再瞬移撞一次 |

非可选单位（不进选人列表，`Fighter: false`）：`子弹`、`分身`、`无下限`、`紫弹`。

颜色绑在 **kind** 上，不绑槽位。新 kind 要在 `internal/web/app.js` 的 `KIND_COLORS` 和 `style.css` 的 `--kind-*` 各加一项，否则会落到近战的青色。

筑墙者的墙长在 Go 里是 `wallLen`，前端瞄准虚线是 `WALL_GUIDE_LEN`，改一边必须改另一边。

## 加一只新球

1. 在 `character/` 新建 `名字.go`，`package character`。
2. `init()` 里 `unit.Register`。要出现在选人页就必须 `Fighter: true`、`Role: unit.RoleFighter`。
3. 实现 `unit.Actor`：`Handle(ctx, ev)` 里根据事件发指令，不要改别人的内存。
4. 需要子弹 / 分身 / 别的随从：再 `Register` 一个 `Fighter: false` 的 kind，用 `Spawn` 拉出来。`Spawn` **不能**生成 fighter。
5. 前端补颜色。有新 FX 名字就在 `app.js` 里接；没有的话引擎仍会跑，只是看不见特效。
6. 新文件会随 `_ "xqdj/character"` 自动编进去，不用改 `main.go`。

最小骨架：

```go
func init() {
    unit.Register(unit.Spec{
        Kind:    "新球",
        Role:    unit.RoleFighter,
        Radius:  18,
        MaxHP:   100,
        Speed:   160,   // 巡航速度；冲刺后引擎会往回减速
        Vision:  200,   // 感知半径；0 表示 Sense.Nearby 恒为空
        Fighter: true,
    }, func(unit.SpawnInfo) unit.Actor {
        return &新球{}
    })
}
```

角色跑在自己的 goroutine 里。`Handle` 要短、不要阻塞，指令丢进 `ctx.Out` 即可。感知包每帧都有；碰撞 / 撞墙只在发生时送来。

### 指令

| Cmd | 作用 |
| --- | --- |
| `SetVelocity` | 改自己的速度向量 |
| `Damage` | 对 `To` 造成伤害；引擎只让 `role=fighter` 扣血 |
| `Spawn` | 生成非 fighter 单位（子弹、分身等） |
| `Despawn` | 删掉指定单位 |
| `SwapOwned` | 本体与随机己方分身交换位置和速度（受伤时引擎也会自动做一次） |
| `PlaceWall` | 砌一段胶囊墙 |
| `Force` | 给目标加加速度 `(AX,AY)`，引擎做 `v += a·dt`。不是改写速度，快的球仍能撞上 |
| `Teleport` | 把自己挪到 `(X,Y)`，引擎会夹回六边形内 |
| `FX` | 给前端的一次性特效；不参与物理 |

### 事件

- `Sense`：当前时刻、自己、视野内快照。
- `Collision`：与另一单位相撞（法线指向对方）。
- `WallHit`：撞上六边形边界或场上的墙。

## 引擎会替你做的事

这些是协助者改角色或物理时不要随便推翻的约定。

- 平顶六边形，外接圆半径 `HexRadius = 280`。扫掠 CCD，提交后的状态不允许重叠。
- 撞边和撞墙：入射角 = 反射角。战斗机和子弹一样弹。
- 非弹体互撞：沿法线各保留自己的速率，`a.v = n·|va|`，`b.v = −n·|vb|`。不要改回速度均分。
- 战斗机速率高于 `Spec.Speed` 时，每 0.2s 减 10，减到巡航为止。减速发生在消化指令之后，所以冲刺当帧能顶住。
- 血量 100；`HP <= 0` 由引擎移除。只有 `RoleFighter` 吃 `Damage`。
- 受伤打 3 帧 hit-stop（物理 / 时间 / 感知都停）。致死不换位；胜负等到 hit-stop 结束再判。
- 子弹与主人不做 CCD。分身会和本体相撞。只有弹体穿主人，分身不穿。
- 弹体撞上战斗机后 `solid=false`，等角色自己 `Despawn`。
- 战斗机死亡时，所有 `owner == 该 id` 的单位一起消失。
- 墙是胶囊段，人人弹开（含自己、分身、子弹）；只对敌方战斗机按 0.1s CD 结算伤害。
- 胜负：场上 `role=fighter` 只剩 1（或 0 为平局）。现在只有 1v1。

## 前端注意

- 选人按钮只在槽位 / 种类变化时重建。不要每帧清空 DOM，否则点不中。
- 战场图层顺序：`#guides` → `#walls` → `#fx` → `#units`。
- 快照里的 `kinds` 来自 `unit.FighterKinds()`，注册顺序就是按钮顺序。

## 还没做

见 `PLAN.md`。混战 2～6 和两队对战都还没接，先把 1v1 角色和物理做稳。
