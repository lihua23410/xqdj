# 小球对战

本项目以 MIT 授权，见 `LICENSE`。

场地里的观战模拟器。玩家选一份场地，再选单挑的两只战斗机，或团队赛里两边各 1 到 3 只。引擎按 60Hz 扫掠碰撞推进，页面只负责显示。

当前是单机本地服：Go 进程里跑对局，浏览器经 WebSocket 收快照。对局中没有玩家操作、没有网络同步、没有 AI 托管。单挑页是 `/`，团队赛页是 `/team`。

领域用词见 `CONTEXT.md`：正式叫**战斗机**，不叫角色；那截可拆的墙叫**胶囊墙**（「砌墙」已作废）；场边、硬墙、胶囊墙统称**墙**。索敌看**可瞄准**（瞄准优先度），不看是不是战斗机；出伤仍打敌方战斗机和敌方活随从。

## 跑起来

需要 Go 1.22+。

```bash
go run .
```

打开 [http://127.0.0.1:8080](http://127.0.0.1:8080)。默认监听 `:8080`，用 `XQDJ_ADDR` 改：

```bash
XQDJ_ADDR=127.0.0.1:9000 go run .
```

拉不下来依赖时：

```bash
GOPROXY=https://goproxy.cn,direct go run .
```

前端和角色资源都靠 `go:embed` 打进二进制：`internal/web/embed.go` 收 `*.html *.css *.js lib/*.js`（往 `lib/` 加新文件要同时改这行 pattern）；角色包收的是它自己 `//go:embed` 行列到的东西——多数包只 embed `fx`，地慧星还 embed `status`，面灵气还 embed `status faction`，人偶使是 `fx 素材`。

所以改了 `internal/web/` 下的 html/css/js，或任何**被该包 embed 到**的 Go / css / js / png，都要**重启进程**，浏览器再**强制刷新**（页面按包清单异步插 css/js，缓存也要清）。没被 embed 的文件改了不生效，放 `status/` 也不会自动进二进制。

端口被旧进程占住时：

```bash
# Linux / macOS
fuser -k 8080/tcp

# Windows（先看 PID，再杀）
netstat -ano | findstr :8080
taskkill /PID <PID> /F
```

测试：

```bash
go test ./...
```

加球或改了战斗机子包之后，先 `go generate ./character` 再测。加场地子包则 `go generate ./map`。对应的 `all.go` 过期时 `TestAllGoListsEveryPack` 会失败。

无头胜率（所有战斗机两两对打，不含自己打自己；奇数场换槽）：

```bash
go run ./cmd/winrate -n 100 -timeout 90 -o winrate.md
```

| flag | 默认 | 含义 |
| --- | --- | --- |
| `-n` | 100 | 每组对战场次 |
| `-timeout` | 90 | 单场模拟超时秒数，超时计平 |
| `-o` | `winrate.md` | 输出文件 |
| `-j` | `NumCPU×8` | 同时进行的场次数 |
| `-only` | 空 | 只测该战斗机对其余所有球（不含自己打自己） |

`-only` 会先校验名字；报告头部标出范围，但两张矩阵仍按当时全部已注册战斗机输出行列，无关格子是 `—`。种子由对阵双方名字决定，`-n`、`-j` 不影响结果。名单来自 `unit.FighterKinds()`，不写死只数。

## 目录

```
character/<ascii>/       一只可选战斗机一个子包。目录名必须是 ASCII（Go import 不能含中文）
                         逻辑 .go + 可选 fx/ status/ faction/，以及包根静态文件
                         HTTP 按中文 Kind 挂 /ball/<Kind>/
character/character.go   //go:generate go run generate.go
character/generate.go    扫描子目录，写出 all.go（自身带 //go:build ignore）
character/all.go         生成：import 每个子包 + 把各包导出的 Kind* 再导出到 xqdj/character
character/all_test.go    清单过期 / 空 import 会失败
map/<ascii>/             一份可选场地一个子包。目录名必须是 ASCII（Go import 不能含中文）
map/场地.go              //go:generate go run generate.go（包名 场地；不能 package map）
map/generate.go          扫描子目录，写出 all.go（自身带 //go:build ignore）
map/all.go               生成：blank import 每个子包，触发 init 登记
map/all_test.go          清单过期 / 空 import 会失败
internal/unit/           Actor / Cmd / Sense / Look / Pack / Field，sim 与 character 的唯一协议
internal/sim/            物理、对局状态机、CCD、墙
internal/web/            单挑选人页、团队赛页（`team.html`）、战场、引擎级特效（embed）；lib/ 放 gsap 等第三方脚本；sfx/ 是两段缺省音效。不要为新球改这里
cmd/winrate/             无头胜率矩阵
docs/adr/                架构决策（巡航带子、活随从、场地可选、胶囊墙改名、硬墙穿透、索敌看可瞄准、挡视线）
docs/fighters/           战斗机介绍用的外观和雷达图 svg（只覆盖部分球）
main.go                  HTTP :8080（XQDJ_ADDR 可改）、/、/ws、/ball/<Kind>/
CONTEXT.md               领域用词
战斗机.md                可选战斗机名册：外观、特性、五维雷达
PLAN.md                  尚未做的混战（多于两边）
winrate.md               无头对战胜率（生成物，球改过之后要重跑）
```

`main.go` 用 `_ "xqdj/character"` 拉起战斗机子包 `init()`。生产代码里 **`internal/sim` 不要 import `character`**，只走 `internal/unit`。角色包 **只 import `xqdj/internal/unit`**，不要 import `internal/sim` 或 `internal/web`。测试里可以 import `xqdj/character`，用生成的 `KindXxx` 点名某只球。

场地相反：一场必须有场地，所以 `internal/sim` 用 `_ "xqdj/map"` 拉起子包。列表和查找走 `unit.FieldNames` / `unit.LookupField`。场地包同样只 import `xqdj/internal/unit`。

页面经 `/ws` 发 `select` / `side` / `mode` / `field` / `start` / `pause` / `end` / `log`。`select`、`side`、`mode`、`field` 只在选人阶段生效。`select` 不带 `index` 时：单挑是 `SetSlot`（该边换成这一只），团队赛是往该边追加一只（已经 3 只则忽略）。`select` 带 `index`：`kind` 为空则 `RemoveSeat`，否则 `SetSeat`。`side` 用 `count` 改该边人数（`SetSideLen`，夹在 1 到 3）。`mode` 的 `team` 为真走 `UseTeam`，否则 `UseDuel`。`start` 兼作继续。`log` 回一份 `{type:"log", recap, trace}`，不改对局。引擎每帧广播快照。

## 现有战斗机

选人按钮顺序 = `unit.FighterKinds()` = 各包 `init()` 的注册顺序。包的加载顺序是 `all.go` 的 import 顺序，也就是 **目录名字母序**。下表按这个顺序，共 50 只（含靶子）。巡航和视野是 `Spec` 的开局值；对局中改巡航、改视野的不写在这里。行为见 `CONTEXT.md` 和 `战斗机.md`。

| 目录 `character/` | Kind | 巡航 | 视野 |
| --- | --- | ---: | ---: |
| `acausal` | `非因果` | 160 | 9999 |
| `archer` | `弓箭` | 178 | 9999 |
| `axe` | `盾斧` | 165 | 9999 |
| `bloodpact` | `血契` | 170 | 9999 |
| `charcoal` | `炭翁` | 240 | 150 |
| `doppel` | `分身者` | 175 | 0 |
| `doppel_veteran` | `分身者(百战)` | 0 | 9999 |
| `dummy` | `靶子` | 165 | 9999 |
| `echo` | `回声` | 165 | 9999 |
| `engine` | `内燃机` | 120 | 96 |
| `fisher` | `钓鱼佬` | 148 | 9999 |
| `freischutz` | `中转站[Freischutz]` | 160 | 9999 |
| `glitch` | `地慧星` | 165 | 9999 |
| `godfather` | `教父` | 170 | 0 |
| `godfather_youth` | `教父(青年)` | 180 | 9999 |
| `hajimi` | `反重力猫咪` | 178 | 9999 |
| `hammer` | `钉与锤` | 170 | 200 |
| `hero` | `勇者` | 170 | 9999 |
| `integrated` | `原型机_整合` | 148 | 9999 |
| `knight` | `小骑士` | 178 | 9999 |
| `maggot` | `蛆` | 0 | 9999 |
| `meander` | `迷走` | 170 | 0 |
| `melee` | `原型机_近战` | 195 | 185 |
| `menreiki` | `面灵气` | 150 | 9999 |
| `mezhou` | `我与周旋久` | 120 | 9999 |
| `orca` | `大肥鱼` | 130 | 0 |
| `pastor` | `老牧师` | 160 | 0 |
| `patrol` | `巡回` | 155 | 0 |
| `pink` | `粉红色迷人的pink` | 155 | 9999 |
| `piper` | `吹笛人` | 165 | 9999 |
| `prisoner` | `囚徒` | 180 | 9999 |
| `radar` | `雷达` | 160 | 9999 |
| `radar_veteran` | `雷达(百战)` | 160 | 9999 |
| `ranged` | `原型机_远程` | 148 | 9999 |
| `reaper` | `收割者` | 175 | 0 |
| `relay` | `中转站` | 160 | 0 |
| `rmiu` | `R.缪` | 170 | 9999 |
| `slot` | `老虎机` | 150 | 9999 |
| `sot` | `酒翁` | 165 | 9999 |
| `static` | `静电魔球` | 160 | 280 |
| `subpoena` | `传票员` | 170 | 9999 |
| `tachi` | `太刀` | 140 | 9999 |
| `tanyongchen` | `谭泳辰老先生` | 165 | 9999 |
| `tokei` | `时计` | 170 | 9999 |
| `twin` | `无下限术士` | 200 | 9999 |
| `udongein` | `人偶使` | 152 | 9999 |
| `waller` | `筑墙者` | 155 | 9999 |
| `waller_veteran` | `筑墙者(百战)` | 155 | 9999 |
| `warden` | `盾士` | 165 | 9999 |
| `wolf` | `狼人` | 172 | 9999 |

内燃机开局是 1 档（巡航 120、视野 96），升到 6 档是 300 / 216。`rmiu` 的 package 名是 `r缪`。

非可选单位（不进选人列表，`Fighter: false`）：

- 弓箭：`箭`、`齐射箭`
- 血契：`血契弧`、`血契蝠`
- 炭翁：`炭烬`
- 分身者：`分身`、`分身弧`
- 分身者(百战)：`战影`
- 回声：`回声星弹`、`回声碎片`
- 钓鱼佬：`钓鱼佬鱼塘`、`钓鱼佬场鱼塘`、`钓鱼佬鱼`、`钓鱼佬中鱼`、`钓鱼佬大鱼`
- 地慧星：`地慧星残影`、`地慧星斩击`、`地慧星弧`、`地慧星弹`
- 教父：`教父暗杀者`、`教父狙击者`、`教父贩卖者`、`教父狙击弹`、`教父药物`
- 教父(青年)：`教父(青年)手枪弹`、`教父(青年)强化弹`
- 钉与锤：`钉与锤弧`、`钉与锤钉`、`钉与锤人偶`
- 勇者：`村好剑`、`死战士`、`死法师`、`死圣女`、`战士`、`法师`、`圣女`、`史莱姆`、`法师弹`
- 原型机_整合：`原型机_整合弹`
- 小骑士：`小骑士弧`
- 蛆：`蛆宝宝`、`苍蝇`、`苍蝇卵`
- 迷走：`迷走环线`、`迷走点亮`
- 原型机_近战：`原型机_近战弧`
- 面灵气：`面灵气面具1`、`面灵气面具2`、`面灵气面具3`、`面灵气弹`
- 我与周旋久：`周旋久`、`虚空弹`
- 老牧师：`老牧师字`
- 巡回：`巡回标识物`
- 吹笛人：`吹笛人老鼠`
- 囚徒：`囚徒钩爪`、`囚徒囚笼`、`囚徒绞刑架`、`囚徒电椅`
- 雷达(百战)：`大炮击`、`小炮击`
- 原型机_远程：`子弹`
- 收割者：`收割者镰刀`、`收割者收回镰刀`
- 中转站：`中转站随从`、`中转弹`
- R.缪：`缪`、`缪弹`、`缪弧`
- 酒翁：`酒翁酒`、`酒翁酒场`
- 谭泳辰老先生：`铃铛`、`苦无`、`灵狐场域`、`奶符`
- 时计：`时计弧`、`时计时刻`
- 无下限术士：`红半`、`蓝半`、`无下限术士弧`、`无下限弧`、`紫弹`
- 人偶使：`人偶使偶`、`人偶使炸弹`、`人偶使弹`、`人偶使激光`
- 盾士：`盾`、`盾碎片`、`弱化碎片`
- 狼人：`狼人月亮`、`狼人弧`、`狼人撕咬弧`
- 场地演员（同样 `Fighter: false`，不进选人）：`八边形`、`机关房`、`奇美拉`、`大六边形`、`大六边形旋涡`、`火山`、`熔岩`

墙的瞄准虚线长度是 `Look.WallGuide`，和胶囊墙长度用同一个常量。瞬移夹紧用 `unit.HexContains` / 场地的 `Clamp`，不要抄 280。


## 加一份场地

目标：只在 `map/` 下丢一个 ASCII 目录，跑 `go generate ./map`，选人页出现新场地。**不要改** `main.go`、`internal/sim`、`internal/web`。

选人按钮顺序 = `unit.FieldNames()` = 目录字母序（所以 `circle` 在 `hex` 前面）。开局默认按名字找六边形，不是列表第一项。`SetField` 不认识的名字不改当前这份（和 `SetSlot` 一样）。单挑不接受奇美拉。重名登记 panic。

一份场地填 `unit.Field`：名字、`Shape`、Extent、预放墙、可选 `BootKind`。`Shape` 只是登记键，判定由该场地 `RegisterOutline`。现有键：`hex`、`circle`、`square`、`octagon`、`chimera`。新外轮廓在场地包里登记判定。

```
map/maze/
  迷宫.go    init 里 unit.RegisterField(...)
```

现有场地：`hex` 六边形、`circle` 圆、`volcano` 火山、`chamber` 机关房、`octagon` 八边形、`widehex` 大六边形、`chimera` 奇美拉。`UseTeam` 在当前场地是六边形或还没有名字时改选奇美拉。`unit.HexField()` 是无名六边形几何，给测试垫底，不进选人列表。

## 加一只新球

目标：只在 `character/` 下丢一个目录，跑 `go generate ./character`，选人页出现新球。**不要改** `main.go`、`internal/web/app.js`、`internal/web/style.css`、`internal/web/embed.go`、`internal/web/index.html`。

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
- 导出的 `const KindXxx = "..."` 会再导出到 `xqdj/character`。名字必须以 `Kind` 开头、且首字母大写，生成器才收（`generate.go` 按 `ast` 常量声明筛）。战斗机、随从都可以导。

### 2. 建包并注册

`init()` 里先 `unit.NewPack`，再用 **同一个** `p` 去 `Register` 战斗机和它的随从。

```go
p := unit.NewPack(KindNova, assets) // 没有静态文件就传 nil
p.Register(spec, factory)
```

- `NewPack` 的名字必须是这只球的 **Kind 字符串**（中文也行）。`Register` 会把该 `Spec.Look.Base` 填成 `/ball/<Pack名>`。随从和主人共用这个前缀。
- 包名撞了会 `panic("unit: duplicate pack …")`。
- 没有静态文件就传 `nil`，不要写 `//go:embed`。
- 有文件时：

```go
//go:embed fx
var assets embed.FS
```

有 `status/`、`faction/` 就写成 `//go:embed fx status faction`。包根也可以 embed 单文件（钓鱼佬：`//go:embed fx fish.png`）。**被 embed 的路径必须真实存在**，否则编译失败。不要 embed 空目录占位。

派系图标可以在 Go 侧登记，快照会带绝对路径：

```go
p.RegisterFactions([]unit.FactionLook{
    {ID: unit.FactionCyan, File: "faction/qing.png", Color: "#3ec8e0"},
})
```

`File` 相对包根（embed 根），不是相对 `fx/`；引擎拼成 `/ball/<Pack名>/<File>` 随快照的 `packs` 下发。前端 `registerFactions` 仍然可用，两边都会进页面。

`go:embed` 会把点号 / 下划线开头的文件丢掉；生成 pack 文件清单（`unit.Packs`）时同样跳过这类文件名。

### 3. `Spec` 每个字段

```go
unit.Spec{
    Kind:       "新球",           // 全局唯一。撞名 panic("unit: duplicate kind …")
    Role:       unit.RoleFighter, // 见下表
    Radius:     18,
    MaxHP:      100,
    Speed:      160,              // 开局巡航。可用 SetCruise 改
    Vision:     200,              // Sense.Nearby 的半径。0 = 除自己的随从外谁都看不见。可用 SetVision 改
    Fighter:    true,             // true 才进选人列表。Spawn 不能生成 Fighter
    Semi:       false,            // true = 半圆（无下限术士的红半/蓝半）
    PassWalls:  false,            // true = 不撞墙（紫弹、收回镰刀）
    Mortal:     false,            // true = 活随从：吃伤害/治疗，不计入胜负
    BreakWalls: false,            // true = 击中胶囊墙则该截提前消失并穿过；硬墙穿过不拆；场边仍撞
    StartHP:    0,                // 0 = 开局满血 MaxHP。小骑士用 85
    AimPriority: 0,               // 0 = 未写：战斗机 15、活随从 60、其余 0。快照 `aimPriority`；0 不可索敌。可用 SetAimPriority 改
    Nonsolid:    false,           // true = 不实心：不碰、不进 Hittable。战斗机仍进快照
    VisionBlock: false,           // true = 挡视线：挡索敌视线（Seek 跳过被压住连线段的目标），不进感知过滤、不挡物理。可用 VisionBlock 令牌运行时挂摘
    Cruise:      false,           // true = 吃巡航带子。战斗机默认有；活随从要写
    Shell:       false,           // true = 贴身壳：吸收打向主人的伤害，撞碎时给主人 GuardBreak；不撞墙
    Attach:     false,            // true = 每帧贴主人；不挡伤、不碎、不撞墙。只和敌方战斗机、活随从做 CCD
    ArcSpan:    0,                // 扇环张角（弧度）。0 = 不是扇环。整圈用 2π。用 unit.Deg(120)
    ArcInner:   0,                // 扇环内径；外径是 Radius。角色自己填（现有近战是 18 / 20）
    Look:       unit.Look{...},
}
```

`Role`：

| 值 | 用途 |
| --- | --- |
| `RoleFighter` | 可选战斗机。吃伤害、算胜负。选人必须 `Fighter: true` 且这个 Role |
| `RoleProjectile` | 子弹。撞上任何可撞单位后 `solid=false`，等你 `Despawn`。`PassWalls` / `BreakWalls` 的弹保持 solid。不穿主人 |
| `RoleClone` | 分身。和本体相撞，不穿主人。非致死确认伤害时引擎自动换位 |
| `RoleTwin` | 已不用。无下限术士的红半/蓝半是 `RoleMinion` |
| `RoleHelper` | 特效 / 残影 / 斩击 / 鱼塘 / 月亮。`solid=false`，不参与 CCD |
| `RoleMinion` | 活随从。实心、吃伤害（伤害同样要确认），不计入胜负。要 `Mortal: true` |

随从 Kind 是 **全局扁平表**，必须唯一。建议带主人前缀：`地慧星残影`、`地慧星斩击`，不要叫 `残影`。`Fighter` 一律 `false`。

`Role` 是字符串，写错不会 panic，只会静默变成「非战斗机、不吃伤害」。`Fighter: true` 只管选人列表，`RoleFighter` 才管胜负，两件事都要写对。

工厂函数每次生成返回 **新的** Actor，不要复用指针。

```go
func(info unit.SpawnInfo) unit.Actor {
    return &新球{} // info.OwnerID / info.Slot 开局随从要用
}
```

### 4. 实现 `Actor`

```go
type 新球 struct{ hitReadyAt float64 }

func (a *新球) Handle(ctx unit.Context, ev unit.Event) {
    if unit.AcceptHit(ctx, ev) {
        return
    }
    switch e := ev.(type) {
    case unit.Sense:
        // 每帧；被 Stun 时引擎不发
    case unit.Collision:
        // 撞上别人
    case unit.WallHit:
        // 撞场边、硬墙或胶囊墙
    case unit.FactionChanged:
        // 派系刚变
    case unit.WallSlam:
        // 自己的两截墙胶囊重叠并消失
    case unit.Stunned:
        // 眩晕刚挂上
    case unit.Kill:
        // 记到自己头上的击杀
    }
}
```

- 角色跑在自己的 goroutine。`Handle` 要短，**不要阻塞、不要睡、不要改别人的内存**。要做事就 `ctx.Out <- 某条指令`。
- `ctx.ID` 是自己的单位 id，`ctx.Kind` 是自己的 Kind。
- **战斗机必须处理 `IncomingDamage`**。不确认就不会掉血。快捷函数：
  - `unit.AcceptHit(ctx, ev)`：是报价就原样 `ConfirmDamage`，并返回 true（上面骨架那种）。
  - `unit.ConfirmHit(ctx, d)` / `unit.BlockHit(ctx, d)`：自己拆 `IncomingDamage` 时用。
  - 减伤：自己 `ConfirmDamage{Token, UnitID: ctx.ID, Amount: 更小的值}`。`Amount` 只能比报价更小，不能抬高。
  - `IncomingDamage.Speed` 是挨打当时受害者的速率（近战减伤、R.缪的闪避都用这个）。
- `Sense` 每帧都有（hit-stop / `Stun` 期间不发）：`Time`、`Self`（自己的快照）、`Nearby`（视野内；自己的随从始终在内）、`Field`（当前场地）。快照带 `AimPriority`。**索敌**用 `unit.Seek(s)`（默认只挑敌方、瞄准优先度 1–255 越小越先、相同则当下最近；带挡视线标签的墙和单位压住球心连线段的目标跳过，见 CONTEXT.md「挡视线」；场地预放墙不带标签）；额外过滤用 `unit.SeekIf`（同样查视线）；自己扫 Nearby 时用 `unit.Aimable`（不查视线，自己扫的照常隔墙进行）。不要写死 `RoleFighter`。有锁的技能自己决定何时重锁，引擎不另做全局锁。同槽可瞄准（如缪锁己方缪）自己写，不走 `Seek`。**出伤**仍用 `unit.Hittable` / `unit.EnemyTarget`（敌方战斗机或敌方活随从），和可瞄准分开：瞄准优先度 0 仍可能被扫到、撞到。改优先度用 `unit.SetAim(ctx, id, v)`（会填 `From: ctx.ID`），或自己发 `SetAimPriority`。雷达扫射、镰刀刮、墙刮不挑人，只看出伤。朝一个点走、直线被 `Walls` 挡住时用 `unit.Detour`：改朝第一截挡路墙离目标更近的端点外侧，并记住这一侧 0.3 秒；返回的是点，角色自己 `SetVelocity`。每截墙都算，不看挡视线。场边不绕。
- `Collision` 只在相撞时来。`NX,NY` 是从 `Other` 指向自己的法线。`Other` 是对方快照。贴身出伤不要在战斗机的 `Collision` 里 `Damage`，挂 `Attach` 扇环弹，弹自己打人。可用 `unit.RearmAttach` / `unit.SpawnAttach`。
- `WallHit` 撞场边、硬墙、胶囊墙时来，`Kind` 区分三者（`unit.WallEdge` / `WallHard` / `WallCapsule`）。`PassWalls` / `Attach` / `Shell` 的单位不会撞墙。撞墙那一帧会清掉带 `OnWall` 的瞬时 FS。
- `GuardBreak` 只发给壳的 **主人**：`Spec.Shell` 被物理撞碎。`DespawnOwned` 摘壳不发这个。
- `FactionChanged` 在派系变化时来（撞墙轮换和 `MarkFaction` 都会发），`Stun` 拦不住。
- `WallSlam` 发给墙的主人：两截同一主人的墙胶囊重叠，并已从场上拿掉。主人不进眩晕圈。
- `Stunned` 在眩晕刚挂上时发给本人。`Until` 是到点。
- `Kill` 在单位被打死时发给记功对象：一般是击杀者本人；击杀者是别人的弹或随从时，记到它主人头上（owner 链只走一层）。活随从靠 `Mortal` 认。环境致死不发。收不收由角色自己决定。

不要在角色里抄场地半径 `280` 或假定六条场边，用 `unit.HexContains` / `Sense.Field`。`Teleport` 引擎会夹回可进入区域。

### 5. 指令

| Cmd | 作用 |
| --- | --- |
| `SetVelocity` | 改自己的速度向量。身上巡航 FS 的 M ≠ 1 时，写入后再乘 M |
| `SetCruise` | 改自己巡航的基准速率。不改当前速度。快于巡航拉回；慢于巡航但仍在动则沿当前方向以回拉一半往上推；速率为 0 且未持 `Stand` 则立刻回到巡航 |
| `SetVision` | 改自己的感知半径。0 = 看不见场上其它单位（随从仍进感知） |
| `SetAimPriority` | 改瞄准优先度（0–255）。`From` 是改的人，`UnitID` 是被改的。只能改战斗机或活随从；弹、弧、分身改不了、也变不成可瞄准。当前为 1–255 时谁都能改、后写覆盖；当前为 0 时只有自己（`From == UnitID`）能改回来。快捷：`unit.SetAim(ctx, id, v)` |
| `SetArcSpan` | 改自己这发 `Attach` 扇环弹的张角（弧度）。近战用这个跟速度走 |
| `SetRadius` | 改自己的碰撞半径。≤0 被丢掉。花冠长大用这个；快照 `radius` 会跟着变，前端按这个画 |
| `SetHP` | 直接设置当前 / 最大血量，不发治疗特效，置 0 也不会移除单位。活随从动态初始化血量用 |
| `Damage` | 向战斗机或活随从报价伤害。引擎发 `IncomingDamage`，必须 `ConfirmDamage` 才会扣血。带 `MarkKind` 时，真正掉血那一拍才叠状态；格挡 / 吸收不叠。`MarkIcon` 用绝对路径，如 `/ball/地慧星/status/jianhen.png`。确认时 `NoFreeze` 为真，这记不停帧 |
| `ConfirmDamage` | 确认一笔报价；只有 `0 < Amount < 原值` 才生效，`Amount <= 0` 视为不改（取消用 `BlockDamage`） |
| `BlockDamage` | 取消一笔报价 |
| `Spawn` | 生成非 fighter 单位。`Fighter: true` 的 Kind 会被引擎丢掉。填 `OwnerID: ctx.ID`、`Slot: Self.Slot`，死时随从一起消失 |
| `Despawn` | 删掉指定单位 |
| `DespawnOwned` | 按主人 + kind 清掉随从（摘壳不会发 `GuardBreak`） |
| `SwapOwned` | 本体与随机己方分身交换位置和速度（受伤时引擎也会自动做一次）。本体处于 `Stun` 时不换速度 |
| `PlaceWall` | 造一段胶囊墙。`Kind` 填 `ctx.Kind` 以便墙色跟主人；`Hard` / `Square` 造硬墙和方端判定（场地预放用），硬墙永久存在且不可拆。`Life < 0` 一直留到被拆或相撞消失；`HitGap` 是刮伤间隔，0 表示 0.1 秒；`WithOwner` 则主人倒下时这截一起消失。`VisionBlock: true` 时这截挡索敌视线（不改物理、不改感知）；场地预放墙不带这个标签。瞄准虚线用 `Look.WallGuide`（和墙长同一个常量） |
| `SetWallMotion` | 改一截墙绕墙心的转速（弧度每秒，逆时针为正）和墙心平移速度。`Ram > 0` 时平移途中对每个敌方目标结一次这个伤害。两截同一主人、`StunRadius > 0` 的墙胶囊重叠时一起消失，圈里的敌方站死 `StunDur` 秒，主人不晕 |
| `HoldStill` | 到 `Until` 之前不发感知，速度保持为 0。施加时记下当前速度；再施加则刷新时间并重新记。到点把记下的速度还回去 |
| `Pass` | 令牌。`Hold: true` 时与其他单位相撞不改双方速度、也不做位置分离；墙仍弹。`Hold: false` 放下 |
| `Stand` | 令牌。`Hold: true` 时速率为 0 也不往巡航推。虚弱、居合、锁敌、绞刑架用这个。`Hold: false` 放下 |
| `NoFrameFreeze` | 令牌。`Hold: true` 时该单位及其随从造成的伤害不停帧。`Hold: false` 放下 |
| `NoHealthNumbers` | 令牌。`Hold: true` 时不画该单位自己的头顶数字，不跟到随从。`Hold: false` 放下 |
| `Stun` | 令牌。`Hold: true` 时引擎不发 Sense（自身攻击停在冷却）；`IncomingDamage` / 撞墙 / 碰撞 / 派系变化仍到。`Until > 0` 时到点自动放下，`Until = 0` 要自己 `Hold: false` |
| `VisionBlock` | 令牌。`Hold: true` 时该单位挡索敌视线（`Seek` / `SeekIf` 跳过球心连线段被其体积压住的目标）。敌我不分，自家带标签单位也挡自家；不进感知过滤、不挡物理，不走索敌的路径无感。谁都能给任何单位挂摘。`Hold: false` 放下 |
| `Force` | 给目标加加速度，引擎做 `v += (AX,AY)×dt`。不是改写速度，要持续加速得每帧重发；只对实心单位生效。无下限术士的拉/推 |
| `Teleport` | 把自己挪到 `(X,Y)`，引擎会夹回可进入区域 |
| `MarkFaction` | 给战斗机打派系。`Cycle` 时撞墙（非单位）换派系，同一目标 0.2s 一次；`AmpOut`/`AmpIn` 是角色自己给的倍率（0 = 不改）；`Collect` 凑齐四种时按 `Barrage` 的 kind 朝四周各生成一发（速度用该 kind 的 `Spec.Speed`），或按 `BlastRadius` / `BlastDamage` 炸一次 |
| `ClearFactionSeen` | 清空 `Collect` 记录，当前派系仍算已出现 |
| `Heal` | 给战斗机或活随从回血，不超过 MaxHP，不触发 hit-stop。实际增加 ≥ 0.5 才发引擎 `heal` 特效 |
| `StackMark` | 给单位叠一层状态（Kind / Icon 由角色定）。`Delta` 可为负，减到 0 就消失。`Lifetime` > 0 时每一层从叠上去那一拍起活这么多秒，到点由引擎掉这一层；0 表示不过期。`Hidden` 为 true 时只给角色自己的 lookFX 看，不显示在通用状态 HUD；层数与寿命照常 |
| `ClearMarks` | 清掉该单位指定 Kind 的状态；Kind 为空则全清 |
| `AddFS` | 加一条**瞬时 FS**：`DX,DY` 是方向，`BaseSpeed` 是速率。持着时引擎每帧把速度写成所有瞬时 FS 的向量和（巡航带子暂停）。`OnWall: true` 撞墙即清；`ExpiresAt: 0` 不过期。同 `Token` 覆盖 |
| `RemoveFS` | 按 `Token` 撤一条瞬时 FS；撤完自动回到巡航 |
| `AddFSComponent` | 往**巡航 FS** 的分量袋里塞一分量（`Zone` + `Value` + `Token` + `ExpiresAt`）。改巡航；`M` 变化时当前速率按比例跟着乘 |
| `RemoveFSComponent` | 按 `Token` 撤掉一个巡航分量 |
| `SetFSDirection` | 改巡航 FS 的朝向，不改当前速度 |
| `FX` | 给前端的一次性特效；不参与物理。见第 9 节 |

巡航分量（`unit.FSZone*`）的实际合成公式见 `internal/sim/fs.go`：

```
巡航 = Ft × ((基准速率 + ΣDp) × (1 + ΣDt) + ΣFp) × M   // 下限 0
```

`M ≈ 0` 时速度被钳到 0，钳之前把当前朝向写进巡航 FS 的 `dir`，松手后才有方向可恢复。

`Spawn` 之后拿不到新单位的 id。随从要自己在 `Handle` 里记主人、到点 `Despawn`，或由主人 `DespawnOwned`。

### 6. 外观：`Look` 字段（不是文件）

颜色绑在 **kind** 上，不绑槽位。写在 `Spec.Look`，引擎随快照的 `looks` 发给页面。前端按字段画，不要按 Kind 名字在 `app.js` 里分支。

| 字段 | 作用 |
| --- | --- |
| `Color` | CSS 颜色。球、火花、残影都吃这个 |
| `Ghost` | 速率超过这个值时留冲刺残影（220–280 不等）。`0` 关闭 |
| `Trail` | 拖尾残影（子弹常用） |
| `Glow` | 发光 class |
| `VisionRing` | 画视野圈（近战、钉与锤） |
| `WallGuide` | 墙瞄准虚线长度；`0` 不画。筑墙者用来等于墙长 |
| `Ring` | 画成环而不是实心圆（盾） |
| `Overlay` | 画在 `#over`，不被场地轮廓裁切（斩击、紫弹、扇环弹、刑具） |
| `FX` | 常驻皮肤短名列表，如 `[]string{"glitch"}`。见第 8 节 |
| `Base` | **不要手填**。`p.Register` 写成 `/ball/<Pack名>` |
| `ShareHP` | 头顶数字跟同槽战斗机走（红半/蓝半） |

不要往 `Look` 加新字段塞私产。私产走 `FX` 短名 + 包内 css/js。

### 7. 外观文件放哪

页面加载快照里的 `packs`：每个包的 `files` 里 **所有** `.css` / `.js` 都会被插进页面（不限于 `fx/` 子目录）。png 只当静态文件，通过 `/ball/<Kind>/路径` 访问。

| 目录 | 干什么 | HTTP |
| --- | --- | --- |
| `fx/*.css` | 常驻皮肤、一次性特效用到的 class | `/ball/新球/fx/foo.css` |
| `fx/*.js` | 皮肤 tick、`registerShot`、`registerFactions` | `/ball/新球/fx/foo.js` |
| `fx/shot.js` | 约定放一次性特效登记（名字随意，按字母序加载） | 同上 |
| `status/*.png` | 状态图标。Go 里写死 `/ball/新球/status/foo.png` 填进 `MarkIcon` | `/ball/新球/status/foo.png` |
| `faction/*.png` | 派系图。用 `RegisterFactions` 或 `registerFactions` 登记，不要改 `app.js` | `/ball/新球/faction/qing.png` |

包根也能放素材（钓鱼佬 `fish.png`、钉与锤 `诅咒.png`、R.缪 `fx/天杀星刀-阿赖耶识.png`），按同样规则挂到 `/ball/<Kind>/` 下。图标放 `status/` 只是约定，放包根或 `fx/` 也能被访问（钉与锤的【咒】就放在包根）。

`internal/web/style.css` 只保留所有球共用的底：圆、半圆、扇环、通用 `.glow` / `.trail`、受伤数字、墙。新球的颜色、切片、力场、斩击条都不要写进去。扇环的张角/内径/颜色由各包 `Spec` 和快照 `arcSpan` / `arcInner` / `Look.Color` 决定。

### 8. 常驻皮肤（`Look.FX`）

1. 短名只能是 `[A-Za-z0-9_-]+`，例如 `"glitch"`、`"chroma"`、`"glitch-still"`。不要用中文，前端 `fxID` 匹配不到（`.css` 后缀可有可无）。
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

`tick` 的 `ctx` 有 `now`、`scale`、`cx`、`cy`、`units`、`fxRoot`、`spawnGhost`。`guide` 的 `ctx` 有 `scale`、`cx`、`cy`、`units`、`ensureGuide`、`placeSeg`、`seenGuides`。

4. 把短名填进该 kind 的 `Look.FX`。随从可以和主人不同皮肤（残影用 `"glitch-still"`）。
5. **`window.lookFX` 的短名是全局的**，不按包隔离。不要和别的球撞。`Look.FX` 里已经占用：`archer`、`arrow`、`assassin`、`axe`、`bloodpact`、`bloodpact-bat`、`bond`、`cage`、`chair`、`chamber-lock`、`charcoal`、`chroma`、`crescent`、`dealer`、`drug`、`engine`、`fisher`、`fisher-fish`、`fisher-flood`、`fisher-pond`、`gallows`、`glitch`、`glitch-still`、`godfather`、`hammer`、`hammer-doll`、`hero`、`hero-friend`、`hero-relic`、`hero-slime`、`integrated`、`integrated-shot`、`knight`、`knight-arc`、`maggot`、`maggot-baby`、`maggot-egg`、`maggot-fly`、`mask1`、`mask2`、`mask3`、`meander`、`meander-lit`、`meander-ring`、`mezhou-me`、`mezhou-void`、`mezhou-zhou`、`minion`、`miu`、`miu-arc`、`miu-shot`、`moon`、`nail`、`ningyushi`、`ningyushi-beam`、`ningyushi-bomb`、`ningyushi-doll`、`ningyushi-shot`、`pastor`、`pastor-glyph`、`piper`、`prisoner`、`pull`、`push`、`radar`、`radar-shell`、`radar-shell-sm`、`radar-vet`、`rat`、`reaper`、`relay`、`relay-node`、`relay-shot`、`rocket`、`shadow`、`sickle`、`slash`、`sniper`、`sot`、`sot-stain`、`sot-wine`、`subpoena`、`tachi`、`tan-bell`、`tan-domain`、`tan-kunai`、`tan-milk`、`tanyongchen`、`ticket`、`tokei`、`tokei-moment`、`veteran`、`volcano-crater`、`volcano-lava`、`vortex`、`wolf`、`youth`、`youth-shot`。一次性特效的名字按包隔离，皮肤短名不会。
6. 卸皮肤时引擎会调 `unmount`。只在「class 被移除」时调（换 kind / 换皮肤）；单位被删或切回选人页不会调，所以不要留下必须靠 `unmount` 才清得掉的常驻子节点。

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

在 `fx/shot.js`（经典脚本，不要 `type=module`）：

```js
arena.registerShot("shot", (fx, ctx) => {
  const { x, y, kind } = ctx; // 已经是屏幕坐标
  arena.spawnFx("fx-flash", x, y, kind);
  arena.burst(x, y, kind, 8);
});
```

`registerShot` 用 `document.currentScript` 推出 `/ball/新球`，所以 **必须由该包的 js 调用**，不要在 `app.js` 里登记。两个包都可以叫 `"shot"`，不会互踩。包内脚本也别写顶层 `const` / `function` 重名（那是全局词法绑定，会和别的包撞）。

前端脚本加载是异步的：第一帧可能钩子还没挂上。引擎对未知名字会退化为 `spawnFx("fx-"+name, …)`，所以尽量自己登记；没登记又没 css 动画的名字，元素会因为等不到 `animationend` 而留在 DOM 里。

`window.arena` 提供：`lookOf`、`kindColor`、`screenPos`、`spawnFx`、`burst`、`spawnGhostFrom`、`registerShot`、`registerFactions`、`registerOwnHurt`，以及声音的 `SOUND_MODE`、`voice`、`audio`、`playAudio`（见 [音效](#音效)）。包内脚本不要写顶层 `const arena`，会盖住 `window.arena`。

`spawnFx(cls, x, y, kind, extra = {})`：`extra` 是 CSS 变量，例如 `{ "--ang": "1rad" }`。元素在 `animationend` 后删掉，css 里要有动画。

**不要占用引擎已有的 `FX.Name`**（`app.js` 的 `ENGINE_FX`，会走引擎分支，你的 `registerShot` 不会被调用）：

`hurt`、`heal`、`swap`、`wall-spawn`、`wall-fade`、`wall`、`impact`、`hit`、`faction`

小骑士瞬移用 `"blink"`，不要叫 `"swap"`（`SwapOwned` 已经发引擎的 `swap`）。墙的出现 / 消失 / 碰撞火花不用角色发，引擎会发。

`shot` 和 `scream`、`attack`、`gear`、`cruise`、`verse`、`carve`、`revive` 是「声音即时通道」的名字（`app.js` 的 `IMMEDIATE_AUDIO`）：钩子会在快照到达时先被调一次，`ctx.audioOnly` 为真、坐标是 0；按帧画的那一遍会跳过它们。拿这些名字画气泡 / 贴图，会画在球场左上角外面且看不见（导出的视频里又是正常的，因为导出没有 WS）。要画东西就自己起个名字。

### 10. 派系图标

逻辑仍用引擎指令 `MarkFaction`（青 / 红 / 紫 / 苍 这四个 id 写在 `unit.Faction*`）。**图和颜色是角色私产**：

1. png 放 `faction/`。
2. Go 里 `p.RegisterFactions`（推荐，快照直接带路径），或 `fx/faction.js`：

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

它会重写 `character/all.go`：import 每个含非测试 `.go` 的 ASCII 子包（import 会触发它们的 `init()`），并把各包导出的 `Kind*` 再导出成 `character.KindXxx`。**不要手改 all.go**。

某个子目录导不出任何 `Kind*` 常量时，那条 import 没人用，`all.go` 会编译失败；两个包导出同名 `Kind*` 会 const 重复；目录里出现两个 package 会让生成器直接报错退出。

然后：

```bash
go test ./...
```

`character/all_test.go`：子目录有 `.go` 但 `all.go` 没 import → 失败；`all.go` import 了空目录 → 失败。两个测试只比 import 路径，不校验 `Kind*` 常量清单是否最新。

### 12. 不要做的事

- 改 `internal/web/*`、`main.go` 来加载新 css / 新图标 / 新 `if (kind === "新球")`。
- 在 `app.js` 里写死 `/faction/*.png` 或 Kind 名字。
- 角色 `import` `internal/sim`。
- 用中文当目录名或 import 路径。
- 随从 Kind 叫一个太泛的名字（`子弹` 已被远程占用；新子弹请 `新球弹` 这种）。
- 瞬移 / 场判断抄数字 280，或假定场地是六边形。
- 战斗机不处理 `IncomingDamage`。
- 索敌写死 `Role == unit.RoleFighter`（用 `unit.Seek` / `unit.Aimable`）。
- 出伤写死战斗机、漏掉活随从（用 `unit.Hittable` / `unit.EnemyTarget`）。
- `Spawn` 战斗机 Kind（会被丢掉；选人只走 `Fighter: true` 的注册表）。
- `Look.FX` / `window.lookFX` 短名和别的球撞车。
- `unit.FX.Kind` 填一个没有 `Look.Base` 的字符串（钩子找不到包）。派系闪光除外，那是引擎的 `"faction"`。
- 直接 `new Audio(...).play()` 发声，或点开声音却不给 `mode` / `priority`（见 [音效](#音效)）。
- 把音效放进 `internal/web/sfx/`，或在 `app.js` 里按 Kind 挂音效。

### 13. 核对清单

- [ ] `character/<ascii>/` 只有这一份，没有中文副本
- [ ] `package` 名唯一；战斗机 `const KindXxx` 已导出
- [ ] `NewPack(Kind字符串, assets)` 与 `Spec.Kind` 一致
- [ ] 战斗机 `RoleFighter` + `Fighter: true`；随从 `Fighter: false` 且 Kind 带前缀
- [ ] 战斗机 `Handle` 处理了 `IncomingDamage`
- [ ] 索敌走 `unit.Seek` / `unit.Aimable`，没有写死 `RoleFighter`；出伤走 `Hittable` / `EnemyTarget`
- [ ] 需要的 `//go:embed` 目录都存在；没有文件则 `nil`
- [ ] 皮肤短名、`fx/<短名>.css`、`Look.FX` 三者一致，且没和已占用短名撞车
- [ ] 自己发的 `FX.Name` 已 `registerShot`，且不在引擎保留名里
- [ ] 状态图走 `/ball/<Kind>/status/…`；派系图走 `RegisterFactions` / `registerFactions`
- [ ] 音效走 `arena.voice` / `audio` / `playAudio`，重触发是 `stop → reset → play`，mode 与 priority 已显式给
- [ ] 自带受击音已 `arena.registerOwnHurt(Kind)`；新音效的响度已量过并写进 `docs/audio-loudness.md`
- [ ] 包内 `fx/README.md` 记了音效来源与许可
- [ ] 没有硬编码 280 / 六条场边
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
    }, func(info unit.SpawnInfo) unit.Actor {
        return &新球{slot: info.Slot}
    })
}

type 新球 struct {
    slot       int
    hitReadyAt float64
}

func (a *新球) Handle(ctx unit.Context, ev unit.Event) {
    if unit.AcceptHit(ctx, ev) {
        return
    }
    switch e := ev.(type) {
    case unit.Sense:
        if t := unit.Seek(e); t != nil {
            _ = t // 瞄准 t
        }
    case unit.Collision:
        if !unit.EnemyTarget(e, a.slot) || e.Time < a.hitReadyAt {
            return
        }
        ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: 6}
        a.hitReadyAt = e.Time + 0.1
    }
}
```

带皮肤时：把 `Look` 改成 `Look{Color: "#7ad0ff", FX: []string{"nova"}}`，`NewPack` 传入 embed 的 `fx`，并加上 `fx/nova.css`（以及可选的 `fx/nova.js`、`fx/shot.js`）。然后 `go generate ./character`。

## 音效

声音和特效一样是**角色私产**：文件放包内，播放在包内的 `fx/*.js` 里登记。引擎只提供三样东西——一个仲裁器（`arena.voice`）、两段全局缺省音效（受击 / 碰撞），以及导出时的音轨采集。约定见 `docs/adr/0019-export-audio-by-event-log.md`、`docs/adr/0020-sound-mode-and-priority.md`、`docs/adr/0021-reset-playhead-on-retrigger.md`。

### 1. 三种来源

| 来源 | 文件 | 什么时候响 |
| --- | --- | --- |
| 角色自带 | `character/<ascii>/fx/*.wav` / `*.mp3` | 角色自己的 `unit.FX` 上，`registerShot` 的钩子里播 |
| 缺省受击音 | `internal/web/sfx/default-hit.wav` | 引擎发 `hurt`（掉血）时；该 kind 调过 `arena.registerOwnHurt` 就不放 |
| 缺省碰撞音 | `internal/web/sfx/default-impact.wav` | 引擎发 `impact`（身体互撞）/ `hit`（弹体命中）时；总是放 |

`hurt` 与 `impact`/`hit` 是两回事：掉血才响受击音，撞上才响碰撞音，互相不替代。角色自带受击音时，在 `fx/*.js` 顶部登记一次，缺省受击音就对它彻底闭嘴（见 `character/pastor/fx/shot.js`）：

```js
arena.registerOwnHurt("老牧师");   // 参数是 Kind 字符串；活随从自带受击音也要登记它自己的 Kind
```

两段缺省音效的参数写死在 `app.js` 的 `DEFAULT_SFX` 里：

| | mode | priority | 冷却 | 声部 |
| --- | --- | ---: | ---: | ---: |
| `hit` | 1（抢占） | 200 | 45ms | 3 |
| `impact` | 0（不抢占） | 210 | 45ms | 3 |

45ms 内的第二下直接丢掉，3 个 `Audio` 轮着用——连击时不至于糊成一片。

### 2. 所有声音都走仲裁器

```js
// 建一次，反复用（推荐）
const v = arena.voice(new Audio(pack + "/fx/attack.wav"), {
  mode: arena.SOUND_MODE.PREEMPT,
  priority: 120,
  volume: 1.0,      // 可选，默认 1
  loop: false,      // 可选
});

// 重触发：停 → 归零 → 播
v.stop();
v.reset();
v.play();

// 一次性播完拉倒（内部建完就 play，不复用）
arena.playAudio(src, { mode: arena.SOUND_MODE.NORMAL, priority: 200 });
```

`new Audio(...).play()` 直接播不行：它绕开仲裁，该被抢占时停不下来；导出页也接不到这次播放，成片里就少这一声。

### 3. mode 与 priority

| mode | 值 | 行为 |
| --- | ---: | --- |
| `NORMAL` | 0 | 默认。不抢占：场上有 `PREEMPT` 在播就让路；同级之间共存 |
| `PREEMPT` | 1 | 起播时停掉正在播的（`IMMUNE` 除外）；但若正在播的有效优先级比自己强，自己让路 |
| `IMMUNE` | 2 | 双向免疫：既不停别人，也不会被 `PREEMPT` 停。有效优先级固定 256 |
| `FORCE` | 3 | 起播停掉**所有**在播的并锁住窗口，窗口内只有后到的 `FORCE` 能接管。有效优先级固定 1 |

`priority` 取 0–255，**数值越小越强，`1` 最强**，`0` 表示“最低”（内部按 256 排）。没写就是 `NORMAL` + `0`。优先级比较看的是“有效优先级”：`IMMUNE` → 256，`FORCE` → 1，其余取 `priority` 的档次。

“正在播”按 `started && !paused && !ended` 算，声音自然播完窗口就关了；暂停、命中停顿、导出页的假时钟共用同一套判断，所以实时与成片听感一致。

**被抢占过的 voice 会带着 `cancelled` 标记，之后 `play()` 一律返回 false——必须 `reset()` 才能再播。** 这就是所有重触发都写成 `stop → reset → play` 的原因。

两个现成的取值参考：地慧星斩击用 `PREEMPT/1`（最强的抢占音）；时计的刻钟与复活用 `IMMUNE`（只管自己不被压，不去压别人），见 `character/tokei/fx/shot.js`。

### 4. 点击之后才能出声

浏览器要求先有用户手势。角色自己做了“静音试播解锁”：第一次 `pointerdown` 时静音播一下就停，之后正常发声（见 `character/glitch/fx/shot.js` 的 `unlock`、`character/wolf/fx/shot.js`）。开场就要响的角色照抄这段。

### 5. 触发时机：`shot` 与 `IMMEDIATE_AUDIO`

快照到达时先按到达顺序把声音播掉，画面仍走 rAF 合并（`app.js` 的 `playSoundFX`）。理由：两发命中挤进同一帧时，后一声会把前一声 `stop` 掉，听起来只剩一下。

- 挂在 `shot` 上的声音（绝大多数）自动享受这个顺序：钩子被调两次，一次 `audioOnly: true`（到达时），一次 `visualOnly: true`（画的时候）。模板（例：`character/ranged/fx/shot.js`）：

```js
arena.registerShot("shot", (fx, ctx) => {
  const audioOnly = !!(ctx && ctx.audioOnly);
  const visualOnly = !!(ctx && ctx.visualOnly);
  if (!visualOnly) { v.stop(); v.reset(); v.play(); }
  if (audioOnly) return;
  // 画东西
});
```

- 挂在别的名字上的声音默认跟画面一起播（合并后）。要它也按到达顺序，得把 FX 名加进 `app.js` 的 `IMMEDIATE_AUDIO`（现在有 `attack`、`gear`、`cruise`、`scream`、`verse`、`carve`、`revive`）——那是引擎侧改动，加之前先确认。

### 6. 重触发与暂停

- 重触发一律 `stop() → reset() → play()`；`reset()` 会把播放头归零。只 `pause + play` 会从上一段的尾部起播，听上去像没声。
- 命中停顿、暂停后用 `pause()` / `play()` 恢复，**不要** `reset()`：那是“接着放”，不是“重来”。
- 循环音别 `reset()`，会从开头重来。

### 7. 响度与音量

- 全部音效统一到 **-15.7 LUFS**（基准 `character/glitch/fx/iai.mp3`），真峰值 ≤ **-1.0 dBFS**；播放音量一律 1.0，不再按文件单独压低。
- 加了新音效，用 `ffmpeg -filter_complex ebur128=peak=true` 量一遍，把结果补进 `docs/audio-loudness.md` 的表。
- 极短音（0.1–0.3 秒）积分响度没有意义，按真峰值归一到 -1.0 dBFS。

### 8. 素材与许可

- 角色自带音效：来源、许可、署名要求写进包内 `fx/README.md`（例：`character/tokei/fx/README.md`、`character/melee/fx/README.md`）。
- 两段缺省音效：`internal/web/sfx/README.md`（Kenney Impact Sounds，CC0 1.0）。
- 只用可商用 / 公共领域素材；CC0 不强制署名，但来源要记下来。

### 9. 导出视频里的声音

导出页不真出声，只把 `play` / `pause` 接住，按帧号记下 `{src, frame, volume, loop, rate, preservesPitch, offset}`，服务端再按 `frame * 800` 采样混成一条轨。所以：

- 用 `arena.voice` / `arena.audio` / `arena.playAudio`，或自己 `new Audio(src)` 再 `play()`——这些都会被记下来。
- **不要**用 Web Audio（`AudioContext`、`AudioBufferSourceNode`）、`<audio autoplay>`、内联 data URI：不进事件表，成片里没有这声。
- 路径必须是页面能取到的 URL（`/ball/<Kind>/fx/…` 或 `/sfx/…`），服务端要按同一份 `unit.BallFS()` 读回来混音。
- 导出页把 `audio.paused` 伪装成 false 并对重复 `play()` 去重：角色 tick 里“没播完就再 play”的写法不会叠音，但也别靠 `paused` 判断状态。

## 引擎会替你做的事

这些是改角色或物理时不要随便推翻的约定。巡航带子的来由见 `docs/adr/0001-cruise-is-a-band.md`；活随从见 `docs/adr/0002-living-minions-take-damage.md`；场地可选见 `docs/adr/0003-field-is-selectable-data.md`；胶囊墙改名见 `docs/adr/0004-masonry-renamed-capsule-wall.md`；硬墙穿透见 `docs/adr/0005-break-walls-pass-through-hard-walls.md`；索敌看可瞄准见 `docs/adr/0006-seek-by-aim-priority.md`；无下限术士两半叠中扣两次见 `docs/adr/0007-twin-halves-stack-hits.md`；挡视线见 `docs/adr/0009-vision-block-is-entity-tag.md`；标记寿命见 `docs/adr/0022-stack-lifetime.md`。

- **场地是数据**。库在 `map/`（包名 `场地`）：一份场地一个 ASCII 子包，`init` 里 `unit.RegisterField`；`go generate ./map` 写出 `all.go`。`unit.Field` 有形状、场心到场边距离和预放墙；库里默认六边形（平顶、`Extent = HexRadius = 280`、无硬墙），另有圆（`Extent = 280`，场心一条硬墙，开局 `x ∈ [-110, 110]`、半宽 6，绕场心顺时针 8 秒一圈；转到固体身上时挤出，并按固体自己的速度弹开）。`Match.SetField` 只在选人阶段生效，开打后忽略；不认识的名字不改当前这份。单挑拒绝奇美拉。`UseTeam` 在当前场地是六边形或名字为空时改选奇美拉。库里另有火山、机关房、八边形、大六边形和奇美拉：轮廓判定是各场地私产（`RegisterOutline` 登记实现），其中八边形截角方形（场心到边 280·cos22.5°），开局由场地演员（BootKind）在三个 120° 均分位砌三段带 `VisionBlock` 的方端硬墙——走 `PlaceWall` 造墙链，不是预放墙（ADR 0009）。大六边形外形同默认六边形，场心到场边 472.5（ADR 0014）；开局两只中心对称的旋涡钉在场心半径 280 的圆上，12 秒逆时针一圈，战斗机、活随从、子弹碰上就送到对面往场心偏一截并把方向倒转 180°（Helper 不送）；场心另有一条同圆的转墙（长 220、粗 6、8 秒顺时针一圈），由场地演员 `PlaceWall{Hard, VisionBlock}` 砌出（ADR 0009），不是预放墙。`NewMatch` 按名字装六边形，没有则列表第一份，登记表空则用 `unit.HexField()` 垫底（不进选人列表）。落点、夹紧、撞边都走 `unit.Field`，角色不要自己算。
- **时间**：`sim.TickHz = 60`，`DT = 1/60`，hit-stop `HitStopFrames = 3`。引擎每 tick 广播一次快照。
- **帧序**（`Match.Tick`）：hit-stop 中只消化指令并递减计数（归零那拍判胜负）；否则 FS 过期 → 同步巡航 → 巡航带子 → 消化指令 → 施加瞬时 FS → 结算碰撞 → 物理推进 → 应用本帧伤害 → 再结算碰撞 → 清掉没打中的附着弹 → 贴壳 / 贴随从 → 会转的硬墙前进一拍并挤出挡路的固体 → 推进时间 → 墙和眩晕过期 → 发快照 → 判胜负。
- **出生**：战斗机在可进入区域随机落点（两个圆心至少隔 `r+R+8`），初速大小 = `Spec.Speed`、方向随机。
- **巡航带子**：快于巡航每 0.2s 减 10（不低于巡航）；慢于巡航但仍在动每 0.2s 加 5（不高于巡航）；速率为 0 且巡航大于 0 时，沿巡航朝向立刻回到巡航，不爬升。持 `Stand`、持瞬时 FS、或巡航 FS 的 `M ≈ 0` 时都不推；只对实心且有巡航带子的单位生效（战斗机默认有；活随从要 `Spec.Cruise`）。拉/推发生在消化指令之前，当帧 `SetVelocity` 能顶住巡航带子，但仍乘当时的 `M`；被瞬时 FS 覆盖时顶不住。
- **撞墙**：入射角 = 反射角，速率不变，场边、硬墙、胶囊墙一样；战斗机和普通子弹一样。例外：`PassWalls` / `Attach` / `Shell` 单位根本不碰墙；`BreakWalls` 的弹撞场边照弹、撞硬墙穿过、撞胶囊墙拆穿。
- **非弹体互撞**：沿法线各保留自己的速率，`a.v = n·|va|`，`b.v = −n·|vb|`。不要改回速度均分。任一方持有 `Pass` 时双方速度都不改、也不做位置分离（`Collision` 仍发）。墙不受 `Pass` 影响。
- **伤害**：血量取自 `Spec.MaxHP`（引擎没有默认值，多数角色填 100）；确认伤害把血打到 ≤0 的那一拍由引擎移除（`SetHP` 置 0 不会移除单位）。`RoleFighter` 和 `Spec.Mortal` 的活随从都吃伤害，也都走 `IncomingDamage` → `ConfirmDamage`：不确认就不掉血。区别在确认之后：活随从不停帧、不换位、不叠 `MarkKind`、不计入胜负。对它造成伤害用 `unit.Hittable`（不实心的不算）。
- **索敌**：不写死战斗机。`Spec.AimPriority` 未写时战斗机 15、活随从 60、其余 0（其余不能改成可瞄准）。快照 `AimPriority`：1–255 可被 `unit.Seek` 挑中（越小越先，相同则当下最近）；0 不可索敌，但 `Nearby` 仍按视野收录，出伤仍看 `Hittable`。1–255 时谁都能 `SetAimPriority`，后写覆盖；改成 0 后只有自己能改回来。观众不画这个数。雷达扫射一类不挑人的出伤不走 Seek。**挡视线**是实体标签（`Spec.VisionBlock` / `VisionBlock` 令牌 / `PlaceWall.VisionBlock`）：`Seek` / `SeekIf` 把「球心连线被带标签实体压住」的目标跳过，墙按胶囊、单位按圆、相切算挡、敌我不分；只挡索敌这一层，不进感知过滤、不挡物理，Nearby 自扫与碰撞出伤照常隔墙进行；场地预放墙不带标签，永远透明。没有带标签实体的场地 `Seek` 行为逐字节不变。
- **hit-stop**：战斗机受伤打 3 帧（物理 / 时间 / 感知都停，指令仍消化）。打活随从不停帧。致死当拍不再换位；胜负在帧末判定，若这拍进入 hit-stop 就顺延到计数归零那一拍。持有 `NoFrameFreeze` 的单位（及其随从）造成的伤害不停帧。确认伤害带 `NoFreeze` 的那一记也不停帧。
- **壳**：`Spec.Shell` 的壳贴住主人。主人身上有壳时，`IncomingDamage` 整包被吸收：壳碎、主人收 `GuardBreak`、伤害为 0。
- **`Stun`**：期间不发 `Sense`；碰撞、撞墙、`IncomingDamage`、`FactionChanged`、`GuardBreak` 仍到。`Until > 0` 的眩晕到点自动解除。
- **子弹**：普通弹撞上任何可撞单位都会 `v=0`、`solid=false`，等角色自己 `Despawn`。弹体和**同槽任何单位**（含主人）都不做 CCD。`PassWalls` / `BreakWalls` 的弹保持 solid，穿过时还会把对方沿反法线推到自己速率。分身和本体是普通实心对，照撞。
- **墙有三种**：场边（轮廓）、硬墙（方端、不可拆）、胶囊墙（圆头、可被 `BreakWalls` 拆）。人人弹开（含自己、分身、子弹）；对敌方战斗机和活随从按 0.1s CD 结算伤害（同一目标一跳，主人和同槽不吃）。`BreakWalls` 的弹击中胶囊墙则拆穿该截，击中硬墙穿过且不拆（ADR 0005），场边仍挡住。硬墙和预放的胶囊墙看起来是冷灰，战斗机造的墙跟主人色。
- **死亡带走随从**：单位死亡时，所有 `owner == 该 id` 的单位一起消失。
- **胜负**：某一边的 `role=fighter` 清零就结束。两边都没了是平局。`winner` 是还活着那一边的开局名单，用顿号接起来；`winnerId` 是该边还活着的一只战斗机。单挑名单只有一只，看起来就是那只的名字。引擎本身没有超时；超时计平只存在于 `cmd/winrate`。槽位只有左右两边。

## 前端注意

- 单挑选人页分场地按钮和左右两个槽。团队赛页每边最多 3 只：点角色加入，点名单移出。按钮只在槽位 / 种类 / 场地变化时重建（`renderLobby` 比 key）。不要每帧清空 DOM，否则点不中。
- 半径 0 的单位不画球。快照 `noHealthNumbers` 藏该单位自己的头顶数字。`Look.ShareHP` 的单位头顶数字跟同槽战斗机走。
- 战场图层顺序：`#guides` → `#walls` → `#fx` → `#units`；`Look.Overlay` 的单位在 `#over`（`#hex` 的兄弟节点，`z-index` 更高，普通单位盖不过它）。
- 快照顶层字段：`type`（恒 `"state"`）、`phase`、`slots`、`roster`（开局名单）、`team`、`kinds`、`looks`、`packs`、`winner` / `winnerId`、`time` / `seq`、`hexRadius`（缩放用的场心到场边距离，名字是历史遗留）、`fields` / `field` / `fieldShape`、`hitStop`、`logReady`、`units`、`walls`、`effects`。
- `kinds` 来自 `unit.FighterKinds()`，注册顺序就是按钮顺序。`looks` 是全部已注册 kind 的外观。`packs` 是各角色 embed 里的静态文件清单（`files` / `factions`），页面据此加载 `/ball/<Kind>/` 下的 css/js 和派系图：`files` 里所有 `.css` / `.js` 都会插进页面，其它后缀只当静态文件。
- `effects` 只在运行阶段每帧清空；选人 / 暂停 / 结束阶段会重复发上一帧的数组。
- 引擎级特效名见第 9 节的保留名单（其中 `wall` 在保留集合里但没有渲染分支，等于什么都不画）。一次性特效按包隔离，常驻皮肤短名全局共享。

## 还没做

见 `PLAN.md`。团队赛已经接上：`/team`，每边 1 到 3 只。多于两边的混战还没有，`slots` 仍是左右两边。
