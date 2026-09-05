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

前端文件用 `go:embed` 打进二进制。改了 `internal/web/` 下的 html/css/js 必须重启进程，浏览器再强制刷新。改了 `character/<包>/` 下的 Go / `fx` / `status` / `faction` 同样要重启（资源打进包的 embed）。端口被旧进程占住时：

```bash
fuser -k 8080/tcp
go run .
```

测试：

```bash
go test ./...
```

加球或改了子包之后，先 `go generate ./character` 再测。`character/all.go` 过期时 `TestAllGoListsEveryPack` 会失败。

## 目录

```
character/<ascii>/  一只可选球一个子包。目录名必须是 ASCII（Go import 不能含中文）
                    逻辑 .go + 可选 fx/ status/ faction/
                    HTTP 按中文 Kind 挂 /ball/<Kind>/
character/character.go   //go:generate go run generate.go
character/generate.go    扫描子目录，写出 all.go（//go:build ignore）
character/all.go         生成的空白 import + 把各包导出的 Kind* 再导出到 xqdj/character
character/all_test.go    清单过期 / 空 import 会失败
internal/unit/      Actor / Cmd / Sense / Look / Pack，sim 与 character 的唯一协议
internal/sim/       物理、对局状态机、CCD、墙
internal/web/       选人页 + 战场 + 引擎级特效（embed）。不要为新球改这里
main.go             HTTP :8080、/ws、/ball/<Kind>/
PLAN.md             尚未做的混战 / 分队
```

`main.go` 用 `_ "xqdj/character"` 拉起所有子包 `init()`。生产代码里 **`internal/sim` 不要 import `character`**，只走 `internal/unit`。角色包 **只 import `xqdj/internal/unit`**，不要 import `internal/sim` 或 `internal/web`。测试里可以 import `xqdj/character`，用生成的 `KindXxx` 点名某只球。

## 现有角色

| Kind | 巡航 | 视野 | 行为概要 |
| --- | --- | --- | --- |
| `原型机_近战` | 195 | 185 | 敌方进入视野边沿时朝对方最多转 15°，沿当前朝向 +140。身前扇环弹伤 7，0.1s 命中后再挂；张角 90°–180° 跟减伤比走（无减伤 90°，满 25% 才 180°）。挨打时按超出巡航的速度减伤，最多 25% |
| `原型机_远程` | 148 | 9999 | 每 1.6s 射一发；子弹速 185、伤 9、撞墙 3 次后消失；开火后自身速度反向 |
| `分身者` | 175 | 0 | 本体不出伤；3 个分身各挂 360° 细环弹伤 6，0.1s 命中后再挂。本体挨打且未死时与随机分身换位置 |
| `筑墙者` | 155 | 9999 | 第 2.7 秒起每 2.7s 在与敌方中点砌一段长 155 的胶囊墙，存活 7.5s，伤 3.5 |
| `无下限术士` | 200 | 9999 | 开局裂成红蓝两个半圆；红吸引、蓝推开。两半各挂 90° 扇环弹伤 7。红蓝相撞时朝敌人打一发紫弹（体积约 5.6 倍、速 300、伤 12），穿透敌人可反复结算，穿出六边形后飞出屏幕才消失 |
| `小骑士` | 178 | 9999 | 开局 85 血；身前 120° 扇环弹伤 9。每 6s 瞬移到敌人身边（不出六边形）再朝对方冲撞。放完有 50% 概率 0.5s 后再瞬移撞一次 |
| `盾士` | 165 | 9999 | 贴身盾环挡住伤害；盾碎或自己格挡时炸出碎片。满盾 8 片伤 6，弱化 6 片伤 4。8s 补盾，或累计挨打 40 后补弱盾 |
| `面灵气` | 150 | 9999 | 球色自己渐变，不改别人颜色。开局给自己和敌人各一个派系图标（青/红/紫/苍）。撞墙（不是撞人）时随机换成另一个。对异派系伤害 +15%，受同派系伤害 −15%。自身：红挂 150° 扇环弹伤 8、紫回血（每秒 1）、青射弹、苍加速（同时只生效当前派系）。凑齐四种后朝四周打出青/红/紫/苍四发弹幕 |
| `地慧星` | 165 | 9999 | 天蓝色球，常驻每帧故障切片。身前 60° 扇环弹伤 6 并叠【剑痕】，20% 留下可穿过的静止故障残影，敌人经过残影再叠一层剑痕。受击时无效该次攻击，瞬移到离敌人最远的八角笼角并朝对方打一发标准弹（12s CD，开局可用），原地留残影。每次撞墙给当前速度 +50，可叠加，真正掉血时拉回巡航。20s 起锁定并立刻索敌，球体带 1.5 球径短刀光蓄力 2s（开时播居合音）；2s 刀光拉满全屏、宽 3 倍球径，再 1s 按 1/3x 收宽。动画结束后按瞄准方向、宽 3 倍球径的无限长矩形结算，伤害 `(6+剑痕×2)×(1+残影)`，然后清掉剑痕和残影 |

非可选单位（不进选人列表，`Fighter: false`）：`子弹`、`分身`、`分身弧`、`无下限`、`无下限术士弧`、`无下限弧`、`紫弹`、`青弹`、`面具青`、`面具红`、`面具紫`、`面具苍`、`面灵气弧`、`盾`、`盾碎片`、`弱化碎片`、`地慧星残影`、`地慧星斩击`、`地慧星弧`、`地慧星弹`、`原型机_近战弧`、`小骑士弧`。

墙的瞄准虚线长度是 `Look.WallGuide`，和砌墙长度用同一个常量。瞬移夹紧用 `unit.HexRadius` / `unit.HexContains`，不要抄 280。

现有包的目录（ASCII）和 Kind（中文）对照：

| 目录 `character/` | `package` / 战斗机 Kind |
| --- | --- |
| `melee` | `原型机_近战` |
| `ranged` | `原型机_远程` |
| `doppel` | `分身者` |
| `waller` | `筑墙者` |
| `twin` | `无下限术士` |
| `knight` | `小骑士` |
| `warden` | `盾士` |
| `menreiki` | `面灵气` |
| `glitch` | `地慧星` |

选人按钮顺序 = `unit.FighterKinds()` = 各包 `init()` 的注册顺序。包的加载顺序是 `all.go` 的 import 顺序，也就是 **目录名字母序**。

## 加一只新球

目标：只在 `character/` 下丢一个目录，跑 `go generate ./character`，选人页出现新球。 **不要改** `main.go`、`internal/web/app.js`、`internal/web/style.css`、`internal/web/embed.go`、`internal/web/index.html`。

谁发出的特效谁拥有：角色私产放该包的 `fx/` / `status/` / `faction/`。引擎只处理通用反馈（受伤数字、治疗、换位、墙、碰撞火花）和 HUD 槽位。

### 1. 建 ASCII 目录

```
character/nova/
  新球.go          逻辑；文件名随意，package 建议等于 Kind
  fx/              可选。css / js
  status/          可选。状态图标 png
  faction/         可选。派系图标 png
```

- **目录名必须是 ASCII**（`nova`、`glitch`）。Go 的 import 路径不能含中文；生成器按目录拼 `"xqdj/character/"+dir`。
- 不要再复制一份中文目录（`character/新球/`）。生成器会扫到每个含 `.go` 的子目录，中文路径会直接编译失败。
- `package` 子句可以用中文 Kind：`package 新球`。`all.go` 用这个名字写 `新球.KindXxx`。
- 两个包不能用同一个 `package` 名，否则 `all.go` 里标识符冲突。
- 导出的 `const KindXxx = "..."` 会再导出到 `xqdj/character`。名字必须以 `Kind` 开头、且首字母大写，生成器才收。战斗机、随从都可以导。

### 2. 建包并注册

`init()` 里先 `unit.NewPack`，再用 **同一个** `p` 去 `Register` 战斗机和它的随从。

```go
p := unit.NewPack("新球", assets) // 或 nil
p.Register(spec, factory)
```

- `NewPack` 的名字必须是这只球的 **Kind 字符串**（中文也行）。它会给本包所有 `Look.Base` 填成 `/ball/新球`。随从和主人共用这个前缀。
- 包名撞了会 `panic("unit: duplicate pack …")`。
- 没有静态文件就传 `nil`，不要写 `//go:embed`。
- 有文件时：

```go
//go:embed fx
var assets embed.FS
```

有 `status/`、`faction/` 就写成 `//go:embed fx status faction`。**被 embed 的目录必须真实存在**，否则编译失败。不要 embed 空目录占位。

`go:embed` 会把点号 / 下划线开头的文件丢掉；生成器列出的 pack 文件同样跳过这类名字。

### 3. `Spec` 每个字段

```go
unit.Spec{
    Kind:      "新球",           // 全局唯一。撞名 panic("unit: duplicate kind …")
    Role:      unit.RoleFighter, // 见下表
    Radius:    18,
    MaxHP:     100,
    Speed:     160,              // 巡航速率；引擎在超速时每 0.2s 减 10
    Vision:    200,              // Sense.Nearby 的半径。0 = 谁都看不见
    Fighter:   true,             // true 才进选人列表。Spawn 不能生成 Fighter
    Semi:      false,            // true = 半圆（无下限术士）
    PassWalls: false,            // true = 不撞墙、不撞边（紫弹）
    StartHP:   0,                // 0 = 开局满血 MaxHP。小骑士用 85
    Shell:     false,            // true = 贴身壳，物理撞碎时给主人 GuardBreak
    Attach:    false,            // true = 每帧贴主人；不挡伤、不碎。不撞墙。只和敌方战斗机做 CCD
    ArcSpan:   0,                // 扇环张角（弧度）。0 = 不是扇环。整圈用 2π。用 unit.Deg(120)
    ArcInner:  0,                // 扇环内径；外径是 Radius。角色自己填（现有近战是 18 / 20）
    Look:      unit.Look{...},
}
```

`Role`：

| 值 | 用途 |
| --- | --- |
| `RoleFighter` | 可选战斗机。吃伤害、算胜负。选人必须 `Fighter: true` 且这个 Role |
| `RoleProjectile` | 子弹。撞上战斗机后 `solid=false`，等你 `Despawn`。不穿主人 |
| `RoleClone` | 分身。和本体相撞，不穿主人 |
| `RoleTwin` | 双生体。前端血条跟同槽战斗机走 |
| `RoleHelper` | 特效 / 残影 / 斩击。`solid=false`，不参与 CCD |

随从 Kind 是 **全局扁平表**，必须唯一。建议带主人前缀：`地慧星残影`、`地慧星斩击`，不要叫 `残影`。`Fighter` 一律 `false`。

工厂函数每次生成返回 **新的** Actor，不要复用指针。

```go
func(info unit.SpawnInfo) unit.Actor {
    return &新球{} // info.OwnerID / info.Slot 开局随从要用
}
```

### 4. 实现 `Actor`

```go
type 新球 struct { hitReadyAt float64 }

func (a *新球) Handle(ctx unit.Context, ev unit.Event) {
    if unit.AcceptHit(ctx, ev) {
        return
    }
    switch e := ev.(type) {
    case unit.Sense:
        // 每帧
    case unit.Collision:
        // 撞上别人
    case unit.WallHit:
        // 撞边或场上的墙
    }
}
```

- 角色跑在自己的 goroutine。`Handle` 要短，**不要阻塞、不要睡、不要改别人的内存**。要做事就 `ctx.Out <- 某条指令`。
- `ctx.ID` 是自己的单位 id，`ctx.Kind` 是自己的 Kind。
- **战斗机必须处理 `IncomingDamage`**。不确认就不会掉血。快捷函数：
  - `unit.AcceptHit(ctx, ev)`：是报价就原样 `ConfirmDamage`，并返回 true（上面骨架那种）。
  - `unit.ConfirmHit(ctx, d)` / `unit.BlockHit(ctx, d)`：自己拆 `IncomingDamage` 时用。
  - 减伤：自己 `ConfirmDamage{Token, UnitID: ctx.ID, Amount: 更小的值}`，见近战。
- `Sense` 每帧都有：`Time`、`Self`（自己的快照）、`Nearby`（视野内）。找敌人扫 `Role == unit.RoleFighter` 且 `Slot != Self.Slot`。
- `Collision` 只在相撞时来。`NX,NY` 是指向对方的法线。`Other` 是对方快照。贴身出伤不要在战斗机的 `Collision` 里 `Damage`，挂 `Attach` 扇环弹，弹自己打人。
- `WallHit` 撞六边形边界或场上墙时来（`PassWalls` / `Attach` 的单位不会撞墙）。
- `GuardBreak` 只发给壳的 **主人**：`Spec.Shell` 被物理撞碎。`DespawnOwned` 摘壳不发这个。

不要在角色里抄场地半径 `280`，用 `unit.HexRadius` / `unit.HexContains`。`Teleport` 引擎会夹回六边形内。

### 5. 指令

| Cmd | 作用 |
| --- | --- |
| `SetVelocity` | 改自己的速度向量 |
| `SetArcSpan` | 改自己这发 `Attach` 扇环弹的张角（弧度）。近战用这个跟速度走 |
| `Damage` | 向战斗机报价伤害。引擎发 `IncomingDamage`，必须 `ConfirmDamage` 才会扣血。带 `MarkKind` 时，真正掉血那一拍才叠状态；格挡 / 吸收不叠。`MarkIcon` 用绝对路径，如 `/ball/地慧星/status/jianhen.png` |
| `ConfirmDamage` | 确认一笔报价；`Amount` 可比原值更小 |
| `BlockDamage` | 取消一笔报价 |
| `Spawn` | 生成非 fighter 单位。`Fighter: true` 的 Kind 会被引擎丢掉。填 `OwnerID: ctx.ID`、`Slot: Self.Slot`，死时随从一起消失 |
| `Despawn` | 删掉指定单位 |
| `DespawnOwned` | 按主人 + kind 清掉随从（摘盾不会发 `GuardBreak`） |
| `SwapOwned` | 本体与随机己方分身交换位置和速度（受伤时引擎也会自动做一次） |
| `PlaceWall` | 砌一段胶囊墙。`Kind` 填 `ctx.Kind` 以便墙色跟主人。瞄准虚线用 `Look.WallGuide`（和墙长同一个常量） |
| `Force` | 给目标加加速度 `(AX,AY)`，引擎做 `v += a·dt`。不是改写速度，快的球仍能撞上 |
| `Teleport` | 把自己挪到 `(X,Y)`，引擎会夹回六边形内 |
| `MarkFaction` | 给战斗机打派系。`Cycle` 时撞墙（非单位）换派系；`AmpOut`/`AmpIn` 是角色自己给的倍率（0 = 不改）；`Collect` 凑齐四种时按 `Barrage` 的 kind 朝四周各生成一发（速度用该 kind 的 `Spec.Speed`） |
| `ClearFactionSeen` | 清空 `Collect` 记录，当前派系仍算已出现 |
| `Heal` | 给战斗机回血，不超过 MaxHP，不触发 hit-stop |
| `StackMark` | 给单位叠一层状态（Kind / Icon 由角色定）。`Delta` 可为负 |
| `ClearMarks` | 清掉该单位指定 Kind 的状态；Kind 为空则全清 |
| `FX` | 给前端的一次性特效；不参与物理。见第 8 节 |

`Spawn` 之后拿不到新单位的 id。随从要自己在 `Handle` 里记主人、到点 `Despawn`，或由主人 `DespawnOwned`。

### 6. 外观：`Look` 字段（不是文件）

颜色绑在 **kind** 上，不绑槽位。写在 `Spec.Look`，引擎随快照的 `looks` 发给页面。前端按字段画，不要按 Kind 名字在 `app.js` 里分支。

| 字段 | 作用 |
| --- | --- |
| `Color` | CSS 颜色。球、火花、残影都吃这个 |
| `Ghost` | 速率超过这个值时留冲刺残影（近战 280）。`0` 关闭 |
| `Trail` | 拖尾残影（子弹常用） |
| `Glow` | 发光 class |
| `VisionRing` | 画视野圈（近战） |
| `WallGuide` | 墙瞄准虚线长度；`0` 不画。筑墙者用来等于墙长 |
| `Ring` | 画成环而不是实心圆（盾） |
| `Overlay` | 画在 `#over`，不被六边形裁切（斩击、紫弹、扇环弹） |
| `FX` | 常驻皮肤短名列表，如 `[]string{"glitch"}`。见第 7 节 |
| `Base` | **不要手填**。`p.Register` 写成 `/ball/<Pack名>` |

不要往 `Look` 加新字段塞私产。私产走 `FX` 短名 + 包内 css/js。

### 7. 外观文件放哪

页面加载快照里的 `packs`：每个包的 `files` 里 **所有** `.css` / `.js` 都会被插进页面（不限于 `fx/` 子目录）。png 只当静态文件，通过 `/ball/<Kind>/路径` 访问。

| 目录 | 干什么 | HTTP |
| --- | --- | --- |
| `fx/*.css` | 常驻皮肤、一次性特效用到的 class | `/ball/新球/fx/foo.css` |
| `fx/*.js` | 皮肤 tick、`registerShot`、`registerFactions` | `/ball/新球/fx/foo.js` |
| `fx/shot.js` | 约定放一次性特效登记（名字随意，按字母序加载） | 同上 |
| `status/*.png` | 状态图标。Go 里写死 `/ball/新球/status/foo.png` 填进 `MarkIcon` | `/ball/新球/status/foo.png` |
| `faction/*.png` | 派系图。用 `registerFactions` 登记，不要改 `app.js` | `/ball/新球/faction/qing.png` |

`internal/web/style.css` 只保留所有球共用的底：圆、半圆、扇环、通用 `.glow` / `.trail`、受伤数字、墙。新球的颜色、切片、力场、斩击条都不要写进去。扇环的张角/内径/颜色由各包 `Spec` 和快照 `arcSpan` / `arcInner` / `Look.Color` 决定。

### 8. 常驻皮肤（`Look.FX`）

1. 短名只能是 `[A-Za-z0-9_-]+`，例如 `"glitch"`、`"chroma"`、`"glitch-still"`。不要用中文，前端 `fxID` 匹配不到。
2. 放 `fx/<短名>.css`。页面给单位加 class `look-<短名>`。CSS 请写 `.ball.look-glitch` 这种，不要依赖 Kind 字符串。
3. 需要每帧动：再放 `fx/<短名>.js`（或写在别的 js 里），往全局 `window.lookFX` 挂钩子：

```js
window.lookFX = window.lookFX || {};
window.lookFX.glitch = {
  tick(el, u, ctx) { /* 每帧 */ },
  unmount(el) { /* 皮肤卸掉时清 DOM */ },
  guide(u, ctx) { /* 可选：画 #guides 里的连线 */ },
};
```

`tick` 的 `ctx` 有 `now`、`scale`、`cx`、`cy`、`fxRoot`、`spawnGhost`。`guide` 的 `ctx` 有 `units`、`ensureGuide`、`placeSeg`、`seenGuides`。

4. 把短名填进该 kind 的 `Look.FX`。随从可以和主人不同皮肤（残影用 `"glitch-still"`）。
5. **`window.lookFX` 的短名是全局的**，不按包隔离。不要和别的球撞（现在有 `glitch`、`glitch-still`、`slash`、`chroma`、`pull`、`push`、`bond`）。一次性特效的名字按包隔离，皮肤短名不会。
6. 卸皮肤时引擎会调 `unmount`，不要在 css 里只加 class 却在 js 里留下无法清理的子节点还不实现 `unmount`。

### 9. 一次性特效（`unit.FX` + `registerShot`）

角色自己发：

```go
ctx.Out <- unit.FX{
    Name: "shot",        // 自己起的名字
    Kind: ctx.Kind,      // 必须是本包的某个 kind，前端靠 looks[kind].base 找钩子
    X: sx, Y: sy,
    VX: ux, VY: uy,      // 可选，朝向
    Slot: s.Self.Slot,
    UnitID: ctx.ID,      // 可选
    Amount: 0,           // 可选；引擎 hurt/heal 会填
}
```

在 `fx/shot.js`（经典脚本，不要 type=module）：

```js
arena.registerShot("shot", (fx, ctx) => {
  const { x, y, kind } = ctx; // 已经是屏幕坐标
  arena.spawnFx("fx-flash", x, y, kind);
  arena.burst(x, y, kind, 8);
});
```

`registerShot` 用 `document.currentScript` 推出 `/ball/新球`，所以 **必须由该包的 js 调用**，不要在 `app.js` 里登记。两个包都可以叫 `"shot"`，不会互踩。

前端脚本加载是异步的：第一帧可能钩子还没挂上。引擎对未知名字会退化为 `spawnFx("fx-"+name, …)`，所以尽量自己登记。

`window.arena` 提供：`lookOf`、`kindColor`、`screenPos`、`spawnFx`、`burst`、`spawnGhostFrom`、`registerShot`、`registerFactions`。

`spawnFx(cls, x, y, kind, extra = {})`：`extra` 是 CSS 变量，例如 `{ "--ang": "1rad" }`。元素在 `animationend` 后删掉，css 里要有动画。

**不要占用引擎已有的 `FX.Name`**（会走引擎分支，你的 `registerShot` 不会被调用）：

`hurt`、`heal`、`swap`、`wall-spawn`、`wall-fade`、`wall`、`impact`、`hit`、`faction`

小骑士瞬移用 `"blink"`，不要叫 `"swap"`（`SwapOwned` 已经发引擎的 `swap`）。墙的出现 / 消失 / 碰撞火花不用角色发，引擎会发。

### 10. 派系图标

逻辑仍用引擎指令 `MarkFaction`（青 / 红 / 紫 / 苍 这四个 id 写在 `unit.Faction*`）。**图和颜色是角色私产**：

1. png 放 `faction/`。
2. `fx/faction.js`：

```js
arena.registerFactions([
  { id: "青", file: "faction/qing.png", color: "#3ec8e0" },
  { id: "红", file: "faction/hong.png", color: "#ff3b3b" },
  { id: "紫", file: "faction/zi.png", color: "#b44cff" },
  { id: "苍", file: "faction/cang.png", color: "#8dffb0" },
]);
```

`file` 相对包根（embed 根），不是相对 `fx/`。HUD 槽位和徽章布局在引擎，没登记就不画。数组顺序 = pip 顺序。`id` 撞了会覆盖。

### 11. 生成清单

在仓库根目录：

```bash
go generate ./character
```

它会重写 `character/all.go`：空白 import 每个 ASCII 子包（触发 `init`），并把各包导出的 `Kind*` 再导出成 `character.KindXxx`。 **不要手改 all.go**。

然后：

```bash
go test ./...
```

`character/all_test.go`：子目录有 `.go` 但 `all.go` 没 import → 失败；`all.go` import 了空目录 → 失败。

### 12. 不要做的事

- 改 `internal/web/*`、`main.go` 来加载新 css / 新图标 / 新 `if (kind === "新球")`。
- 在 `app.js` 里写死 `/faction/*.png` 或 Kind 名字。
- 角色 `import` `internal/sim`。
- 用中文当目录名或 import 路径。
- 随从 Kind 叫一个太泛的名字（`子弹` 已被远程占用；新子弹请 `新球弹` 这种）。
- 瞬移 / 场判断抄数字 280。
- 战斗机不处理 `IncomingDamage`。
- `Spawn` 战斗机 Kind（会被丢掉；选人只走 `Fighter: true` 的注册表）。
- `Look.FX` / `window.lookFX` 短名和别的球撞车。
- `unit.FX.Kind` 填一个没有 `Look.Base` 的字符串（钩子找不到包）。派系闪光除外，那是引擎的 `"faction"`。

### 13. 核对清单

- [ ] `character/<ascii>/` 只有这一份，没有中文副本
- [ ] `package` 名唯一；战斗机 `const KindXxx` 已导出
- [ ] `NewPack(Kind字符串, assets)` 与 `Spec.Kind` 一致
- [ ] 战斗机 `RoleFighter` + `Fighter: true`；随从 `Fighter: false` 且 Kind 带前缀
- [ ] 战斗机 `Handle` 处理了 `IncomingDamage`
- [ ] 需要的 `//go:embed` 目录都存在；没有文件则 `nil`
- [ ] 皮肤短名、`fx/<短名>.css`、`Look.FX` 三者一致
- [ ] 自己发的 `FX.Name` 已 `registerShot`，且不在引擎保留名里
- [ ] 状态图走 `/ball/<Kind>/status/…`；派系图走 `registerFactions`
- [ ] `go generate ./character` 且 `go test ./...` 通过
- [ ] 没改 `internal/web` / `main.go`

### 14. 最小骨架（无特效文件）

```go
package 新球

import "xqdj/internal/unit"

const KindNova = "新球"

func init() {
    p := unit.NewPack(KindNova, nil)
    p.Register(unit.Spec{
        Kind:    KindNova,
        Role:    unit.RoleFighter,
        Radius:  18,
        MaxHP:   100,
        Speed:   160,
        Vision:  200,
        Fighter: true,
        Look:    unit.Look{Color: "#7ad0ff"},
    }, func(unit.SpawnInfo) unit.Actor {
        return &新球{}
    })
}

type 新球 struct{ hitReadyAt float64 }

func (a *新球) Handle(ctx unit.Context, ev unit.Event) {
    if unit.AcceptHit(ctx, ev) {
        return
    }
    switch e := ev.(type) {
    case unit.Collision:
        if e.Other.Role != unit.RoleFighter || e.Time < a.hitReadyAt {
            return
        }
        ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: 6}
        a.hitReadyAt = e.Time + 0.1
    }
}
```

带皮肤时：把 `Look` 改成 `Look{Color: "#7ad0ff", FX: []string{"nova"}}`，`NewPack` 传入 embed 的 `fx`，并加上 `fx/nova.css`（以及可选的 `fx/nova.js`、`fx/shot.js`）。然后 `go generate ./character`。

## 引擎会替你做的事

这些是协助者改角色或物理时不要随便推翻的约定。

- 平顶六边形，外接圆半径 `HexRadius = 280`。扫掠 CCD，提交后的状态不允许重叠。
- 撞边和撞墙：入射角 = 反射角。战斗机和子弹一样弹。
- 非弹体互撞：沿法线各保留自己的速率，`a.v = n·|va|`，`b.v = −n·|vb|`。不要改回速度均分。
- 战斗机速率高于 `Spec.Speed` 时，每 0.2s 减 10，减到巡航为止。减速发生在消化指令之后，所以冲刺当帧能顶住。
- 血量默认 100；`HP <= 0` 由引擎移除。只有 `RoleFighter` 吃伤害，且必须确认 `IncomingDamage`。
- 受伤打 3 帧 hit-stop（物理 / 时间 / 感知都停）。致死不换位；胜负等到 hit-stop 结束再判。
- 子弹与主人不做 CCD。分身会和本体相撞。只有弹体穿主人，分身不穿。
- 弹体撞上战斗机后 `solid=false`，等角色自己 `Despawn`。
- 战斗机死亡时，所有 `owner == 该 id` 的单位一起消失。
- 墙是胶囊段，人人弹开（含自己、分身、子弹）；只对敌方战斗机按 0.1s CD 结算伤害。
- 胜负：场上 `role=fighter` 只剩 1（或 0 为平局）。现在只有 1v1。

## 前端注意

- 选人按钮只在槽位 / 种类变化时重建。不要每帧清空 DOM，否则点不中。
- 战场图层顺序：`#guides` → `#walls` → `#fx` → `#units`；`Look.Overlay` 的单位在 `#over`。
- 快照里的 `kinds` 来自 `unit.FighterKinds()`，注册顺序就是按钮顺序。`looks` 是全部已注册 kind 的外观。`packs` 是各角色 embed 里的静态文件清单，页面用来加载 `/ball/<Kind>/` 下的 css/js。

## 还没做

见 `PLAN.md`。混战 2～6 和两队对战都还没接，先把 1v1 角色和物理做稳。
