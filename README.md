# 小球对战

场地里的 1v1 观战模拟器。玩家选一份场地和两只战斗机，引擎按 60Hz 扫掠碰撞推进，页面只负责显示。

当前是单机本地服：Go 进程里跑对局，浏览器经 WebSocket 收快照。没有玩家操作、没有网络同步、没有 AI 托管。

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

`-only` 会先校验名字；报告头部标出范围，但两张矩阵仍按全部 21 只球输出行列，无关格子是 `—`。种子由对阵双方名字决定，`-n`、`-j` 不影响结果。

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
internal/web/            选人页（含场地）+ 战场 + 引擎级特效（embed）；lib/ 放 gsap 等第三方脚本。不要为新球改这里
cmd/winrate/             无头胜率矩阵
docs/adr/                架构决策（巡航带子、活随从、场地可选、胶囊墙改名、硬墙穿透、索敌看可瞄准）
docs/fighters/           战斗机介绍用的外观和雷达图 svg（只覆盖部分球）
main.go                  HTTP :8080（XQDJ_ADDR 可改）、/、/ws、/ball/<Kind>/
CONTEXT.md               领域用词
战斗机.md                可选战斗机名册：外观、特性、五维雷达
PLAN.md                  尚未做的混战 / 分队
winrate.md               无头对战胜率（生成物，球改过之后要重跑）
```

`main.go` 用 `_ "xqdj/character"` 拉起战斗机子包 `init()`。生产代码里 **`internal/sim` 不要 import `character`**，只走 `internal/unit`。角色包 **只 import `xqdj/internal/unit`**，不要 import `internal/sim` 或 `internal/web`。测试里可以 import `xqdj/character`，用生成的 `KindXxx` 点名某只球。

场地相反：一场必须有场地，所以 `internal/sim` 用 `_ "xqdj/map"` 拉起子包。列表和查找走 `unit.FieldNames` / `unit.LookupField`。场地包同样只 import `xqdj/internal/unit`。

页面经 `/ws` 发 `select` / `field` / `start` / `pause` / `end`。`select` 和 `field` 只在选人阶段生效（`SetSlot` / `SetField` 里判断 `phase`）；`start` 兼作继续。引擎每帧广播快照。

## 现有战斗机

选人按钮顺序 = `unit.FighterKinds()` = 各包 `init()` 的注册顺序。包的加载顺序是 `all.go` 的 import 顺序，也就是 **目录名字母序**。下表按这个顺序。

| Kind | 巡航 | 视野 | 行为概要 |
| --- | --- | --- | --- |
| `盾斧` | 165 | 9999 | 灰球 r=18。平时盾/红盾：只对见过的来源生效，来源落在正面 300° 内时盾伤 ×0.5、红盾 ×0.25。前方 ≈24.67、120° 摸到人 → 切剑并蓄力二连斩（各 5；红剑且非斧态每段 +2），打中战斗机能量 +1（上限 2）；第一套后以 350 突刺 1.5s，碰到人伤 3 接第二套二连斩，没碰到就结束。连招后 Stand 1s 灌瓶（1 点：白→黄、空→白；2 点：黄），0.5s 后黄瓶转红：红盾→红剑→红斧，红斧巡航 90。红斧索敌 104／120°，大解 18、追解 15×2、超解旋 15、超解扇 30、波 15×3（60° 扇形分 3 环），每段前转向敌人；超解第 3 波后红态与瓶全掉、巡航回 165 并切回盾 |
| `分身者` | 175 | 0 | 本体不出伤，r=18、100 血、视野 0。开局首帧沿自速方向 48 处按 120° 放 3 个分身（r=18、1 血、速 175，初速 = 自速各旋 120°）；分身各挂内外径 18/20 的 360° 细环（Overlay），碰敌方战斗机或活随从伤 6 即消失，0.1s 后再挂。本体挨打且未死时引擎与随机己方分身互换位置（速度一起换，`Stun` 时不换） |
| `靶子` | 165 | 9999 | 测试用蓝球（#4aa3ff）。r=18、开局满血 99999。`Handle` 只做 `AcceptHit` 全额确认：不出伤、不改巡航/视野、不转向，撞墙照弹。用来量别的球的输出与生存 |
| `内燃机` | 120→300 | 96→216 | 档位 1–6：热力条按时间自涨（1–4 档 2s、5 档 4s、6 档 6s，虚弱时停），满则升档、6 档满则击破。撞墙只弹开。升档只 `SetCruise`/`SetVision`，不写当前速度。普通攻击自带 0.5s 节拍，就绪即跳、没人也进 CD；伤 0/0/1/1/2/3。击破后虚弱 2s：速度置 0、节拍不停但不扣血、档位外观保持变暗；第 1 秒按 6 档视野 216 圈打 30，2s 回 1 档并沿击破前朝向以 120 起步。全程持 `NoFrameFreeze` |
| `钓鱼佬` | 148（空军加速：148/222/296…） | 9999 | 不转向。开局在可进入区域随机铺 3 口塘（r=42）。踩塘站住钓 3s（钓时挨打 ×0.5）；走路巡航 = 148×(1+0.5×空军次数)。第二次空军清掉全部开局塘，场心铺 r=320 一口盖满场地，此后场内都算踩塘续竿。第三次空军起每竿投 10 次。鱼撞敌伤 `clamp(y×0.5,1,26)` 后变惰性；y≥50 的先扛着进高速碰撞：巡航 380、撞人伤 `clamp(y×0.5,4,22)`、同目标 0.35s CD，8s 到点甩出。鱼持 `Pass`，弹 3 次后惰性滑行 |
| `地慧星` | 165（撞墙 +50，掉血回 165） | 9999 | 天蓝色球，常驻故障切片。身前 60° 扇环弹伤 6 并叠【剑痕】；20% 留可穿过的静止残影，敌人经过再叠一层剑痕。受击时闪避就绪就无效该次攻击，随机瞬移到可行走点并朝对方打一发弹（r=12、速 600、伤 9），12s CD、开局可用，原地留残影；CD 内则确认伤害并清掉撞墙加速。撞场边或硬墙把当前速率 +50 写进巡航（蓄力中延后、速率为 0 丢弃），真正掉血时巡航与速度一起拍回 165。20s 起锁定并立刻索敌，球体带短刀光蓄力 2s（开时播居合音）；判定是宽 5 倍球直径的无限长矩形，伤 (6+剑痕×2)×(1+残影)，刀线上的敌方战斗机和活随从都结算，命中清被打中目标的剑痕和全部残影、CD 20s，未命中 10s。刀光单位活 3s |
| `教父` | 170 | 0 | 不转向，球自己不出伤。第 1/2/3 秒从场边各冒一只活随从（r=14、巡航 150），之后缺编用一只 2.5s 补人钟；最多三只、种类不重复。暗杀者 15 血、视野 140、2 次冲刺：索敌可瞄准，持 `Pass` 以 640 冲 280 伤 7，同一目标一次只结算 1 次，撞墙或走满后 1s CD。狙击者 14 血、视野 9999，瞄 7s 后朝索敌目标当下位置射一发穿胶囊墙的弹（r=5、速 900、伤 20）。贩卖者 10 血、视野 24，每 5s 丢一包药、丢完 2 包退场，自己不捡、也不索敌；教父捡回 7，活随从捡回 10 并 +1 攻击次数 |
| `钉与锤` | 170 | 200 | 紫球。120° 近战弧伤 7，命中即消失、0.1s 再挂；弧消失那一帧若锤击就绪（5s CD）就把最近可命中单位击退 350、眩晕 2s，并横扫 56+敌半径内 8 伤。每 8s 射五寸钉（r=5、速 500、伤 5、穿单位与胶囊墙，第 3 次撞墙消失），命中叠【咒】、7 层爆 22；命中非人偶目标时以 600 沿钉向钉出、撞墙即停、定身 1.5s。16s 起每 16s 放 3 枚绕敌 56 环行 3s 再飞出的钉。30s 放 3 个替身人偶（血量 = 敌人当时一半，持 Pass 穿过单位，出场 10s 后消失），打人偶的伤转给敌人 |
| `小骑士` | 178 | 9999 | 开局 85 血（满血 100），深红球、冲刺残影阈值 250。身前 120° 扇环弹伤 9（内 18 外 20，0.1s 再挂）。每 6s 瞬移到索敌目标侧面留 8px 空隙的可行走点（挑不到就落场心）再朝对方以 350 冲撞，之后被巡航带子拉回 178。放完 50% 概率 0.5s 后再撞一次，否则等 6s |
| `原型机_近战` | 195 | 185 | 不转向、不改巡航。可瞄准进入视野那一瞬朝对方最多转 15°，沿转向后方向把速率写成当下 +140（一次性，离开再进才再触发）。身前扇环弹伤 7：外径 20、内径 18，命中即散、0.1s 后重挂，张角 = 90°+90°×减伤比，按弹速率（= 主人速度）逐帧重算。挨任何伤害按超出 195 的速率减伤：每超 150 得 5%，超 750 封顶 25% |
| `面灵气` | 150 | 9999 | 开局给自己和敌方各一派系。三张面具顺时针绕球转（2s 一圈，苍叶 1.2s），各自每 120° 装填一次，碰上打 3.5（红莲 ×1.35、体积也变大）。青岚每 60° 射一发、之后每 360°（速 300）。苍叶巡航 250，撞墙、撞人或切到苍叶时朝对方最多转 90°、不加速。紫岩消普通子弹、每秒回 1。1 勾 10s：异派系伤害 ×1.1、同派系挨打 ×0.9。2 勾起自己撞墙换派系；敌人凑齐四色按最后一种结算（红莲面具伤、青岚眩晕 2s、苍叶破甲 4、紫岩 5s 内挨打的 50% 转给敌人）。3 勾（2 勾后 20s）：异派系 ×1.5、同派系挨打 ×0.75，凑齐后重新累计 |
| `囚徒` | 180（触电 360） | 9999 | 不转向，撞墙弹开。开局无钩爪：第一次撞场边才钉（硬墙、胶囊墙都不钉），之后每次撞场边换位置。链长 350：短则巡航弹墙，到 350 绷直绕钩爪转、速度大小不变，首次绷直按切向反向锁向。链和球都不出伤。开局等 5s，之后每 12s 落囚笼/绞刑架/电椅各一（0.3s 前摇、存在 10s，三个落点含索敌目标当下位置且彼此 ≥110）。囚笼八截胶囊墙（内圈 48、粗 6）伤 2、0.1s 一跳；绞刑架 r=16.5 碰一次即 `Stun`（不发 Sense），每 0.4s 伤 1，直到这具消失；电椅 r=16.5 每 0.5s 放 3–6 道折线伤 5（同目标 0.5s 一次），打中锁链则触电 360、5s。球横向黑白条纹，锈铁链节 |
| `雷达` | 160 | 9999 | 不转向，撞墙弹开。开局朝正北挂一条从球心伸出的绿细线激光：判定宽 4、长 = 自己的视野（9999）、只从当前朝向一侧伸出、穿墙（墙只弹雷达自己），顺时针 5.5s 一圈、不跟速度绑。扫到敌方战斗机或敌方活随从打 7，同一目标 0.3s CD；分身、弹、墙都不扣 |
| `原型机_远程` | 148 | 9999 | 不转向，撞墙弹开。有索敌目标才射，每 1.6s 一发（第一帧就绪）；子弹 r=6、速 185、伤 9，撞墙第 3 次当帧消失，打中或碰到不可致伤的单位也消失。开火后自身朝反方向、按当下速率（静止时 148）反向 |
| `收割者` | 175 | 0 | 不转向。撞场边或硬墙时在朝墙一侧 18 处钉一把 2D 暗刃洋红新月镰刀（r=18、伤 6、每把 1s CD，打中不消失、同一处能叠；出过伤整把变鲜红）。钉点用这次撞墙的位置，不沿用上一拍记着的坐标。硬墙在转则刀跟着转；一进入收回就离开墙。胶囊墙不钉。场上钉着的镰刀到 10 把时，那一批全部放大到 36，从静止每秒加速 40 飞回本体（穿墙，原处立刻腾空可再钉），碰到收割者消失。出过伤的刀碰到本体回 3 |
| `R.缪` | 170（击杀 +100/只，半血 +100） | 9999 | 开局周身 56 处放 5 只白缪。本体与缪同形，每帧跟血最高的缪互换血量、位置、速度和标记，致死一击改判给替身。互相也打敌人：身前 60° 扇环伤 5+速度差/20、CD 0.35s；每 12s 对敌三连弹（r=5、速 500、伤 5+速度差/20，隔 0.15s，活 3s）；缪每 6s 以 640 突刺 0.4s，撞墙撞人即断。打掉任一只缪巡航 +100 并继承它的加速，半血再 +100。挨打时按速率差闪避。5% 概率开局成天杀星缪：不换位，每 5s 斩一只缪 120 伤 |
| `无下限术士` | 200 | 9999 | 不转向。战斗机半径 0、不实心、瞄准优先度 0、开局持 `NoHealthNumbers`，钉在出生点；计入胜负的生命在战斗机上。开局裂成红半/蓝半两只活随从（生命 9999 只为不消失、瞄准优先度 15），两半都按 200 直线飞、撞墙弹开；红半吃巡航带子，蓝半不吃。红 `Force` 拉 +300、蓝推 −100，跳过弹体与同槽与不实心。两半各挂 90° 扇环弹伤 7，打中即消、0.1s 后重挂。红蓝相撞时朝敌人打一发紫弹（r=32、速 300、伤 12、开火 CD 0.5），穿墙穿透、每 0.1s 才再结算一次，出场地后 `Abs(x)` 或 `Abs(y)` > 640 才消失。打中任何一半把同一笔（来源仍是敌人）转给战斗机并停帧。同一记打到两半扣两次 |
| `人偶使` | 152 | 9999 | 金黄色球。灵力上限 5，每 0.8s +1（技能锁定期不回灵力、也不能再放技）。挨打报价 ≤10 点时耗 `伤害/10` 灵力把该次伤害减半，不足则清空灵力并吃 ×1.25。5 槽手牌（牌堆 20）。普攻不耗灵力：前方 54 处扇形 120°、6 段各 1.4；或 72×36 矩形 9.8 并推开 600（定身 1.5s）。技能耗 1：前方扇形 100°、7 段各 1.12，或掷出人偶射击 6 发各 1.4。特殊技 [22] 命待命人偶追击、[26] 放人偶椭圆（各锁 0.8s），打出同名符卡当场放符卡特殊并升 1 级。符卡耗 1（战符 2 / 蓬莱 4，需有能量），伤害 ×(级+5)×1.4。充能：造成/承受每段、每耗 1 灵力各 +0.1。开机即持 `NoFrameFreeze` |
| `筑墙者` | 155 | 9999 | 第 2.7s 起每 2.7s 造一段胶囊墙：取与索敌目标中点、垂直连线，长 155、粗 6，存活 7.5s、伤 3.5（0.1s 一跳，打敌方战斗机与活随从）。视野内没有可瞄准就不造，钟点顺延 |
| `筑墙者(百战)` | 155 | 9999 | 砖色裂纹圆，不转向，球自己不出伤。自己的胶囊墙不满两堵就砌，长 155、粗 6、无寿命、刮伤 3（0.3s），1.27s 一堵。没有自己的墙时墙心在场地轮廓里随机、朝向随机。有一堵时第二堵墙心钉在索敌目标的后方（沿第一堵墙心到目标的直线，背离第一堵墙，距目标中心 77.5），朝向随机；墙心须在轮廓内，没有可瞄准、两点重合或后方在轮廓外则钟保持就绪，下一拍能放就立刻放。满两堵对转（每秒最多 90°，转到法线与墙心连线重合；先到的停住等另一堵）后沿这条法线对冲（速率 300，加速度 500）。胶囊重叠则两堵消失。冲撞途中每堵墙对每个敌方目标一次伤 8。相撞中心半径为墙长的一半，圈内的敌方眩晕 2.8 秒：速度保持为 0，到点还回晕前速度。相撞那一瞬在相撞中心炸出一个和眩晕圈同大的砖色圆环。筑墙者(百战)自己不晕。主人倒下墙一起消失 |
| `盾士` | 165 | 9999 | 开局在自身位置挂 `盾`（r=20、`Shell`）贴身壳：`IncomingDamage` 一律 `BlockHit` 并炸出 8 片，物理撞碎（`GuardBreak`）则炸出 6 片。每 8s 重挂：壳还在就先摘壳补炸 6 片弱化碎片再挂满盾；累计确认承受 40 点伤害时同样摘壳→炸 6 片→满盾→清计数。碎片 r=6、速 90–280 随机，撞墙第 2 次或碰到任意可打目标即消失；满盾碎片伤 6，弱化碎片伤 4 |
| `狼人` | 172 | 9999 | 开局在可进入区域随机挂月亮 r=34。自己或敌人进入月亮（边沿触发）推进月相 0→7 取模；到 4（满月）进入撕咬：`Stand` 站定 0.1s 锁敌，再持 `Pass` 以 640 冲墙，共 5 次（每次撞墙后 `Stand` 0.35s 重锁），结束月相归 0 并按锁定方向对墙镜面反射以 172 起飞。常态 `Attach` 弧 110°／内径 18／半径 24／伤 8／CD 0.1s；撕咬时换 150°／伤 13／CD 0.3s |

非可选单位（不进选人列表，`Fighter: false`）：`子弹`、`分身`、`分身弧`、`缪`、`缪弹`、`缪弧`、`人偶使偶`、`人偶使弹`、`人偶使激光`、`人偶使炸弹`、`原型机_近战弧`、`囚徒囚笼`、`囚徒电椅`、`囚徒绞刑架`、`囚徒钩爪`、`地慧星弧`、`地慧星弹`、`地慧星斩击`、`地慧星残影`、`小骑士弧`、`弱化碎片`、`收割者收回镰刀`、`收割者镰刀`、`教父暗杀者`、`教父狙击弹`、`教父狙击者`、`教父药物`、`教父贩卖者`、`红半`、`蓝半`、`无下限弧`、`无下限术士弧`、`狼人弧`、`狼人撕咬弧`、`狼人月亮`、`盾`、`盾碎片`、`紫弹`、`面灵气弹`、`面灵气面具1`、`面灵气面具2`、`面灵气面具3`、`钓鱼佬中鱼`、`钓鱼佬场鱼塘`、`钓鱼佬大鱼`、`钓鱼佬鱼`、`钓鱼佬鱼塘`、`钉与锤人偶`、`钉与锤弧`、`钉与锤钉`。

墙的瞄准虚线长度是 `Look.WallGuide`，和胶囊墙长度用同一个常量。瞬移夹紧用 `unit.HexContains` / 场地的 `Clamp`，不要抄 280。

现有包的目录（ASCII）和 Kind（中文）对照，按 `all.go` import / 选人按钮顺序：

| 目录 `character/` | `package` / 战斗机 Kind |
| --- | --- |
| `axe` | `盾斧` |
| `doppel` | `分身者` |
| `dummy` | `靶子` |
| `engine` | `内燃机` |
| `fisher` | `钓鱼佬` |
| `glitch` | `地慧星` |
| `godfather` | `教父` |
| `hammer` | `钉与锤` |
| `knight` | `小骑士` |
| `melee` | `原型机_近战` |
| `menreiki` | `面灵气` |
| `prisoner` | `囚徒` |
| `radar` | `雷达` |
| `ranged` | `原型机_远程` |
| `reaper` | `收割者` |
| `rmiu` | `r缪`（Kind 是 `R.缪`） |
| `twin` | `无下限术士` |
| `udongein` | `人偶使` |
| `waller` | `筑墙者` |
| `waller_veteran` | `筑墙者(百战)` |
| `warden` | `盾士` |
| `wolf` | `狼人` |

## 加一份场地

目标：只在 `map/` 下丢一个 ASCII 目录，跑 `go generate ./map`，选人页出现新场地。**不要改** `main.go`、`internal/sim`、`internal/web`。

选人按钮顺序 = `unit.FieldNames()` = 目录字母序（所以 `circle` 在 `hex` 前面）。开局默认按名字找六边形，不是列表第一项。`SetField` 不认识的名字不改当前这份（和 `SetSlot` 一样）。重名登记 panic。

一份场地包只能填 `unit.Field`：名字、现有形状（`hex` / `circle`）、Extent、预放墙。新外轮廓要改 unit / sim / 页面，不是丢包能解决的。

```
map/maze/
  迷宫.go    init 里 unit.RegisterField(...)
```

现有两份：`map/hex`（`package 六边形`，`unit.HexField()`）、`map/circle`（`package 圆`，`unit.CircleField()`）。

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
- `Sense` 每帧都有（hit-stop / `Stun` 期间不发）：`Time`、`Self`（自己的快照）、`Nearby`（视野内；自己的随从始终在内）、`Field`（当前场地）。快照带 `AimPriority`。**索敌**用 `unit.Seek(s)`（默认只挑敌方、瞄准优先度 1–255 越小越先、相同则当下最近）；额外过滤用 `unit.SeekIf`；自己扫 Nearby 时用 `unit.Aimable`。不要写死 `RoleFighter`。有锁的技能自己决定何时重锁，引擎不另做全局锁。同槽可瞄准（如缪锁己方缪）自己写，不走 `Seek`。**出伤**仍用 `unit.Hittable` / `unit.EnemyTarget`（敌方战斗机或敌方活随从），和可瞄准分开：瞄准优先度 0 仍可能被扫到、撞到。改优先度用 `unit.SetAim(ctx, id, v)`（会填 `From: ctx.ID`），或自己发 `SetAimPriority`。雷达扫射、镰刀刮、墙刮不挑人，只看出伤。
- `Collision` 只在相撞时来。`NX,NY` 是从 `Other` 指向自己的法线。`Other` 是对方快照。贴身出伤不要在战斗机的 `Collision` 里 `Damage`，挂 `Attach` 扇环弹，弹自己打人。可用 `unit.RearmAttach` / `unit.SpawnAttach`。
- `WallHit` 撞场边、硬墙、胶囊墙时来，`Kind` 区分三者（`unit.WallEdge` / `WallHard` / `WallCapsule`）。`PassWalls` / `Attach` / `Shell` 的单位不会撞墙。撞墙那一帧会清掉带 `OnWall` 的瞬时 FS。
- `GuardBreak` 只发给壳的 **主人**：`Spec.Shell` 被物理撞碎。`DespawnOwned` 摘壳不发这个。
- `FactionChanged` 在派系变化时来（撞墙轮换和 `MarkFaction` 都会发），`Stun` 拦不住。

不要在角色里抄场地半径 `280` 或假定六条场边，用 `unit.HexContains` / `Sense.Field`。`Teleport` 引擎会夹回可进入区域。

### 5. 指令

| Cmd | 作用 |
| --- | --- |
| `SetVelocity` | 改自己的速度向量 |
| `SetCruise` | 改自己巡航的基准速率。不改当前速度。快于巡航拉回；慢于巡航但仍在动则沿当前方向以回拉一半往上推；速率为 0 且未持 `Stand` 则立刻回到巡航 |
| `SetVision` | 改自己的感知半径。0 = 看不见场上其它单位（随从仍进感知） |
| `SetAimPriority` | 改瞄准优先度（0–255）。`From` 是改的人，`UnitID` 是被改的。只能改战斗机或活随从；弹、弧、分身改不了、也变不成可瞄准。当前为 1–255 时谁都能改、后写覆盖；当前为 0 时只有自己（`From == UnitID`）能改回来。快捷：`unit.SetAim(ctx, id, v)` |
| `SetArcSpan` | 改自己这发 `Attach` 扇环弹的张角（弧度）。近战用这个跟速度走 |
| `SetRadius` | 改自己的碰撞半径。≤0 被丢掉。花冠长大用这个；快照 `radius` 会跟着变，前端按这个画 |
| `SetHP` | 直接设置当前 / 最大血量，不发治疗特效，置 0 也不会移除单位。活随从动态初始化血量用 |
| `Damage` | 向战斗机或活随从报价伤害。引擎发 `IncomingDamage`，必须 `ConfirmDamage` 才会扣血。带 `MarkKind` 时，真正掉血那一拍才叠状态；格挡 / 吸收不叠。`MarkIcon` 用绝对路径，如 `/ball/地慧星/status/jianhen.png` |
| `ConfirmDamage` | 确认一笔报价；只有 `0 < Amount < 原值` 才生效，`Amount <= 0` 视为不改（取消用 `BlockDamage`） |
| `BlockDamage` | 取消一笔报价 |
| `Spawn` | 生成非 fighter 单位。`Fighter: true` 的 Kind 会被引擎丢掉。填 `OwnerID: ctx.ID`、`Slot: Self.Slot`，死时随从一起消失 |
| `Despawn` | 删掉指定单位 |
| `DespawnOwned` | 按主人 + kind 清掉随从（摘壳不会发 `GuardBreak`） |
| `SwapOwned` | 本体与随机己方分身交换位置和速度（受伤时引擎也会自动做一次）。本体处于 `Stun` 时不换速度 |
| `PlaceWall` | 造一段胶囊墙。`Kind` 填 `ctx.Kind` 以便墙色跟主人；`Hard` / `Square` 造硬墙和方端判定（场地预放用），硬墙永久存在且不可拆。`Life < 0` 一直留到被拆或相撞消失；`HitGap` 是刮伤间隔，0 表示 0.1 秒；`WithOwner` 则主人倒下时这截一起消失。瞄准虚线用 `Look.WallGuide`（和墙长同一个常量） |
| `SetWallMotion` | 改一截墙绕墙心的转速（弧度每秒，逆时针为正）和墙心平移速度。`Ram > 0` 时平移途中对每个敌方目标结一次这个伤害。两截同一主人、`StunRadius > 0` 的墙胶囊重叠时一起消失，圈里的敌方站死 `StunDur` 秒，主人不晕 |
| `HoldStill` | 到 `Until` 之前不发感知，速度保持为 0。施加时记下当前速度；再施加则刷新时间并重新记。到点把记下的速度还回去 |
| `Pass` | 令牌。`Hold: true` 时与其他单位相撞不改双方速度、也不做位置分离；墙仍弹。`Hold: false` 放下 |
| `Stand` | 令牌。`Hold: true` 时速率为 0 也不往巡航推。虚弱、居合、锁敌、绞刑架用这个。`Hold: false` 放下 |
| `NoFrameFreeze` | 令牌。`Hold: true` 时该单位及其随从造成的伤害不停帧。`Hold: false` 放下 |
| `NoHealthNumbers` | 令牌。`Hold: true` 时不画该单位自己的头顶数字，不跟到随从。`Hold: false` 放下 |
| `Stun` | 令牌。`Hold: true` 时引擎不发 Sense（自身攻击停在冷却）；`IncomingDamage` / 撞墙 / 碰撞 / 派系变化仍到。`Until > 0` 时到点自动放下，`Until = 0` 要自己 `Hold: false` |
| `Force` | 给目标加加速度，引擎做 `v += (AX,AY)×dt`。不是改写速度，要持续加速得每帧重发；只对实心单位生效。无下限术士的拉/推 |
| `Teleport` | 把自己挪到 `(X,Y)`，引擎会夹回可进入区域 |
| `MarkFaction` | 给战斗机打派系。`Cycle` 时撞墙（非单位）换派系，同一目标 0.2s 一次；`AmpOut`/`AmpIn` 是角色自己给的倍率（0 = 不改）；`Collect` 凑齐四种时按 `Barrage` 的 kind 朝四周各生成一发（速度用该 kind 的 `Spec.Speed`），或按 `BlastRadius` / `BlastDamage` 炸一次 |
| `ClearFactionSeen` | 清空 `Collect` 记录，当前派系仍算已出现 |
| `Heal` | 给战斗机或活随从回血，不超过 MaxHP，不触发 hit-stop。实际增加 ≥ 0.5 才发引擎 `heal` 特效 |
| `StackMark` | 给单位叠一层状态（Kind / Icon 由角色定）。`Delta` 可为负，减到 0 就消失 |
| `ClearMarks` | 清掉该单位指定 Kind 的状态；Kind 为空则全清 |
| `AddFS` | 加一条**瞬时 FS**：`DX,DY` 是方向，`BaseSpeed` 是速率。持着时引擎每帧把速度写成所有瞬时 FS 的向量和（巡航带子暂停）。`OnWall: true` 撞墙即清；`ExpiresAt: 0` 不过期。同 `Token` 覆盖 |
| `RemoveFS` | 按 `Token` 撤一条瞬时 FS；撤完自动回到巡航 |
| `AddFSComponent` | 往**巡航 FS** 的分量袋里塞一分量（`Zone` + `Value` + `Token` + `ExpiresAt`），改的是巡航而不是当前速度 |
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
5. **`window.lookFX` 的短名是全局的**，不按包隔离。不要和别的球撞。现在已占用：`assassin`、`axe`、`bond`、`cage`、`chair`、`chroma`、`crescent`、`dealer`、`drug`、`engine`、`fisher`、`fisher-fish`、`fisher-flood`、`fisher-pond`、`gallows`、`glitch`、`glitch-still`、`godfather`、`hammer`、`hammer-doll`、`mask1`、`mask2`、`mask3`、`minion`、`miu`、`miu-arc`、`miu-shot`、`moon`、`nail`、`ningyushi`、`ningyushi-beam`、`ningyushi-bomb`、`ningyushi-doll`、`ningyushi-shot`、`prisoner`、`pull`、`push`、`radar`、`reaper`、`sickle`、`slash`、`sniper`、`wolf`。一次性特效的名字按包隔离，皮肤短名不会。
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

`window.arena` 提供：`lookOf`、`kindColor`、`screenPos`、`spawnFx`、`burst`、`spawnGhostFrom`、`registerShot`、`registerFactions`。包内脚本不要写顶层 `const arena`，会盖住 `window.arena`。

`spawnFx(cls, x, y, kind, extra = {})`：`extra` 是 CSS 变量，例如 `{ "--ang": "1rad" }`。元素在 `animationend` 后删掉，css 里要有动画。

**不要占用引擎已有的 `FX.Name`**（`app.js` 的 `ENGINE_FX`，会走引擎分支，你的 `registerShot` 不会被调用）：

`hurt`、`heal`、`swap`、`wall-spawn`、`wall-fade`、`wall`、`impact`、`hit`、`faction`

小骑士瞬移用 `"blink"`，不要叫 `"swap"`（`SwapOwned` 已经发引擎的 `swap`）。墙的出现 / 消失 / 碰撞火花不用角色发，引擎会发。

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

## 引擎会替你做的事

这些是改角色或物理时不要随便推翻的约定。巡航带子的来由见 `docs/adr/0001-cruise-is-a-band.md`；活随从见 `docs/adr/0002-living-minions-take-damage.md`；场地可选见 `docs/adr/0003-field-is-selectable-data.md`；胶囊墙改名见 `docs/adr/0004-masonry-renamed-capsule-wall.md`；硬墙穿透见 `docs/adr/0005-break-walls-pass-through-hard-walls.md`；索敌看可瞄准见 `docs/adr/0006-seek-by-aim-priority.md`；无下限术士两半叠中扣两次见 `docs/adr/0007-twin-halves-stack-hits.md`。

- **场地是数据**。库在 `map/`（包名 `场地`）：一份场地一个 ASCII 子包，`init` 里 `unit.RegisterField`；`go generate ./map` 写出 `all.go`。`unit.Field` 有形状、场心到场边距离和预放墙；库里默认六边形（平顶、`Extent = HexRadius = 280`、无硬墙），另有圆（`Extent = 280`，场心一条硬墙，开局 `x ∈ [-110, 110]`、半宽 6，绕场心顺时针 8 秒一圈；转到固体身上时挤出，并按固体自己的速度弹开）。`Match.SetField` 只在选人阶段生效，开打后忽略；不认识的名字不改当前这份。`NewMatch` 按名字装六边形，没有则列表第一份，登记表空则用 `unit.HexField()` 垫底（不进选人列表）。落点、夹紧、撞边都走 `unit.Field`，角色不要自己算。
- **时间**：`sim.TickHz = 60`，`DT = 1/60`，hit-stop `HitStopFrames = 3`。引擎每 tick 广播一次快照。
- **帧序**（`Match.Tick`）：hit-stop 中只消化指令并递减计数（归零那拍判胜负）；否则 FS 过期 → 同步巡航 → 巡航带子 → 消化指令 → 施加瞬时 FS → 结算碰撞 → 物理推进 → 应用本帧伤害 → 再结算碰撞 → 清掉没打中的附着弹 → 贴壳 / 贴随从 → 会转的硬墙前进一拍并挤出挡路的固体 → 推进时间 → 墙和眩晕过期 → 发快照 → 判胜负。
- **出生**：战斗机在可进入区域随机落点（两个圆心至少隔 `r+R+8`），初速大小 = `Spec.Speed`、方向随机。
- **巡航带子**：快于巡航每 0.2s 减 10（不低于巡航）；慢于巡航但仍在动每 0.2s 加 5（不高于巡航）；速率为 0 且巡航大于 0 时，沿巡航朝向立刻回到巡航，不爬升。持 `Stand`、持瞬时 FS、或巡航 FS 的 `M ≈ 0` 时都不推；只对实心且有巡航带子的单位生效（战斗机默认有；活随从要 `Spec.Cruise`）。拉/推发生在消化指令之前，当帧 `SetVelocity` 能顶住；但被瞬时 FS 覆盖时顶不住。
- **撞墙**：入射角 = 反射角，速率不变，场边、硬墙、胶囊墙一样；战斗机和普通子弹一样。例外：`PassWalls` / `Attach` / `Shell` 单位根本不碰墙；`BreakWalls` 的弹撞场边照弹、撞硬墙穿过、撞胶囊墙拆穿。
- **非弹体互撞**：沿法线各保留自己的速率，`a.v = n·|va|`，`b.v = −n·|vb|`。不要改回速度均分。任一方持有 `Pass` 时双方速度都不改、也不做位置分离（`Collision` 仍发）。墙不受 `Pass` 影响。
- **伤害**：血量取自 `Spec.MaxHP`（引擎没有默认值，多数角色填 100）；确认伤害把血打到 ≤0 的那一拍由引擎移除（`SetHP` 置 0 不会移除单位）。`RoleFighter` 和 `Spec.Mortal` 的活随从都吃伤害，也都走 `IncomingDamage` → `ConfirmDamage`：不确认就不掉血。区别在确认之后：活随从不停帧、不换位、不叠 `MarkKind`、不计入胜负。对它造成伤害用 `unit.Hittable`（不实心的不算）。
- **索敌**：不写死战斗机。`Spec.AimPriority` 未写时战斗机 15、活随从 60、其余 0（其余不能改成可瞄准）。快照 `AimPriority`：1–255 可被 `unit.Seek` 挑中（越小越先，相同则当下最近）；0 不可索敌，但 `Nearby` 仍按视野收录，出伤仍看 `Hittable`。1–255 时谁都能 `SetAimPriority`，后写覆盖；改成 0 后只有自己能改回来。观众不画这个数。雷达扫射一类不挑人的出伤不走 Seek。
- **hit-stop**：战斗机受伤打 3 帧（物理 / 时间 / 感知都停，指令仍消化）。打活随从不停帧。致死当拍不再换位；胜负在帧末判定，若这拍进入 hit-stop 就顺延到计数归零那一拍。持有 `NoFrameFreeze` 的单位（及其随从）造成的伤害不停帧。
- **壳**：`Spec.Shell` 的壳贴住主人。主人身上有壳时，`IncomingDamage` 整包被吸收：壳碎、主人收 `GuardBreak`、伤害为 0。
- **`Stun`**：期间不发 `Sense`；碰撞、撞墙、`IncomingDamage`、`FactionChanged`、`GuardBreak` 仍到。`Until > 0` 的眩晕到点自动解除。
- **子弹**：普通弹撞上任何可撞单位都会 `v=0`、`solid=false`，等角色自己 `Despawn`。弹体和**同槽任何单位**（含主人）都不做 CCD。`PassWalls` / `BreakWalls` 的弹保持 solid，穿过时还会把对方沿反法线推到自己速率。分身和本体是普通实心对，照撞。
- **墙有三种**：场边（轮廓）、硬墙（方端、不可拆）、胶囊墙（圆头、可被 `BreakWalls` 拆）。人人弹开（含自己、分身、子弹）；对敌方战斗机和活随从按 0.1s CD 结算伤害（同一目标一跳，主人和同槽不吃）。`BreakWalls` 的弹击中胶囊墙则拆穿该截，击中硬墙穿过且不拆（ADR 0005），场边仍挡住。硬墙和预放的胶囊墙看起来是冷灰，战斗机造的墙跟主人色。
- **死亡带走随从**：单位死亡时，所有 `owner == 该 id` 的单位一起消失。
- **胜负**：场上 `role=fighter` 只剩 1 就是胜者（0 为平局），`winnerId` 也会发给页面。引擎本身没有超时；超时计平只存在于 `cmd/winrate`。现在只有 1v1。

## 前端注意

- 选人页分场地按钮和左右两个槽；按钮只在槽位 / 种类 / 场地变化时重建（`renderLobby` 比 key）。不要每帧清空 DOM，否则点不中。
- 半径 0 的单位不画球。快照 `noHealthNumbers` 藏该单位自己的头顶数字。`Look.ShareHP` 的单位头顶数字跟同槽战斗机走。
- 战场图层顺序：`#guides` → `#walls` → `#fx` → `#units`；`Look.Overlay` 的单位在 `#over`（`#hex` 的兄弟节点，`z-index` 更高，普通单位盖不过它）。
- 快照顶层字段：`type`（恒 `"state"`）、`phase`、`slots`、`kinds`、`looks`、`packs`、`winner` / `winnerId`、`time` / `seq`、`hexRadius`（缩放用的场心到场边距离，名字是历史遗留）、`fields` / `field` / `fieldShape`、`hitStop`、`units`、`walls`、`effects`。
- `kinds` 来自 `unit.FighterKinds()`，注册顺序就是按钮顺序。`looks` 是全部已注册 kind 的外观。`packs` 是各角色 embed 里的静态文件清单（`files` / `factions`），页面据此加载 `/ball/<Kind>/` 下的 css/js 和派系图：`files` 里所有 `.css` / `.js` 都会插进页面，其它后缀只当静态文件。
- `effects` 只在运行阶段每帧清空；选人 / 暂停 / 结束阶段会重复发上一帧的数组。
- 引擎级特效名见第 9 节的保留名单（其中 `wall` 在保留集合里但没有渲染分支，等于什么都不画）。一次性特效按包隔离，常驻皮肤短名全局共享。

## 还没做

见 `PLAN.md`。混战 2～6 和两队对战都还没接，先把 1v1 角色和物理做稳。
