# 新增战斗机「巡回」（character/patrol）

## 机制总览

- **巡回**（战斗机）：平时正常巡航。冷却好了就**朝随机方向丢出「巡回标识物」，随即站定不动**（Stand + 速度清零，弓箭蓄力同款）。
- **标识物去程（弹球式）**：沿随机方向直线飞行，**持续加速**；撞墙/场边按入射角反弹（引擎自动反射），继续加速，**直到撞中第一个敌方目标**——这一撞即出伤，随后立即转入回程。
- **标识物回程**：直线朝巡回当前位置飞（巡回站着不动），**继续加速**，途中撞到的每个敌方目标都出伤（每目标一次）。
- **回归**：标识物回到巡回身边后消失，巡回恢复移动（随机新方向），**5 秒后**再次丢出。
- 主人阵亡时引擎自动回收其随从/弹，无需特判。

## 已核实的引擎事实（实现依据）

- `Pass{Hold:true}`：撞单位不改双方速度、不分离，**但墙/场边仍反弹**（contract.go:349-353）；且 `Collision` 事件在 pass 早退**之前**发出（match.go:2495-2512）→ 持 Pass 的弹既收得到碰撞事件又不会被撞停。
- 巡航带子：速率 0 且未持 Stand 会立刻沿巡航朝向恢复巡航（match.go decelererateLocked）→ 站定用 `Stand{Hold:true}` + `SetVelocity{0,0}`，恢复只需 `Stand{Hold:false}` + 给个新方向（弓箭.go:242-243、259 先例）。
- `Vision: 0` 的战斗机仍能在 `Sense.Nearby` 里看到**自己的随从/弹**（收割者 Vision 0 数镰刀先例，收割者.go:118-137）→ 主人轮询 Nearby 即可感知标识物是否还在场。
- 投射物与主人的 CCD 配对被引擎跳过（match.go:2385-2390）→ 回归判定用 Sense 里自算距离（镰刀 overlap 先例）。
- `unit.NewRNG(stream)`（internal/unit/rng.go:39）做可复现随机方向；`Snapshot` 带 VX/VY/Marks 等所需字段。

## 新增文件

### 1. `character/patrol/巡回.go`（package 巡回）

常量（均可调）：

| 常量 | 初值 | 说明 |
|---|---|---|
| patrolRadius / patrolHP / patrolCruise | 18 / 115 / 155 | 站桩期是活靶，血量略补 |
| markRadius / markVision | 11 / 9999 | 标识物要看得到主人 |
| firstThrowDelay / throwCD | 0.8s / **5.0s** | 首丢小前摇；回归后 5 秒再丢（规格值） |
| markStartSpeed / markAccel | 300 / 320 | 去回程共用；无上限（引擎 CCD 全扫掠，箭速 1024 是常态） |
| markTimeout | 10s | 去程兜底：一直没撞到人也折返，防止主人永久站桩 |
| dmgPerSpeed / dmgMin / dmgMax | 0.014 / 8 / 26 | **伤害=clamp(撞击时速×系数)**，hajimi 弹速率算伤先例——加速久、砸得重，弹球蓄势有意义 |
| hitPaceCD | 0.25s | 标识物两次出伤的最小间隔 |
| catchDist | 半径和+2 | 回归判定 |

注册（同一个 `p := unit.NewPack(KindPatrol, assets)`）：
- `KindPatrol = "巡回"`：RoleFighter/Fighter/Radius/HP/Speed/Vision 0，`Look{Color:"#4a7fb5", FX:["patrol"]}`。
- `KindPatrolMarker = "巡回标识物"`：RoleProjectile/MaxHP 1/Speed 0/Vision 9999，**不 PassWalls**（要撞墙反弹），`Look{Color:"#ffab2e", Glow:true, Trail:true, Overlay:true, FX:["patrolmark"]}`。

**巡回 Actor**（`AcceptHit` 必备）：
- 状态：`waiting bool`、`nextThrowAt float64`、`rng`。
- Sense：`!waiting && s.Time >= nextThrowAt` → `throw()`：`rng` 取随机方向；`ctx.Born(Spawn{KindPatrolMarker, 出膛点=自身半径+markRadius+gap, VX/VY=dir×markStartSpeed, OwnerID, Slot})`；`Stand{Hold:true}` + `SetVelocity{0,0}`；`StackMark{Kind:"patrol-out", Delta:1}`；FX `patrol-throw`（带方向 VX/VY 与坐标，弓箭 emitFace 注释的教训）；`waiting=true`。
- `waiting` 时每帧扫 Nearby 找 `Kind==KindPatrolMarker && OwnerID==ctx.ID`；找不到（被收回）→ 恢复：`Stand{Hold:false}`、`SetFSDirection(随机新向)` + `SetVelocity(新向×patrolCruise)`、`ClearMarks("patrol-out")`、`nextThrowAt = s.Time + throwCD`、`waiting=false`。

**标识物 Actor**：
- 首拍 Sense：`Pass{Hold:true}`（镰刀 boot 先例）。
- 去程（`!returning`）：
  - 每帧 `dt = s.Time - lastT`；`speed += markAccel×dt`；方向取引擎真值 `normalize(Self.VX, VY)`（**撞墙反射自动生效**，远程火箭先例）；`SetVelocity(dir×speed)`。
  - `s.Time - bornAt > markTimeout` → 转回程（不出伤）。
  - `Collision` 且 `unit.EnemyTarget(e, slot)` → `Damage{From: owner, To, Amount: clamp(speed×dmgPerSpeed, min, max)}` + FX `patrol-hit`（X/Y=命中点、VX/VY=撞击朝向）→ 清空回程 struck 表、`returning=true`（引擎反弹/穿体不影响，Pass 保护速度）。
  - `WallHit` → FX `patrol-bounce`（≥0.15s 节流，反弹火花=加速反馈，非装饰）。
- 回程（`returning`）：
  - 每帧 `speed += markAccel×dt`；朝 `ownerOf(s, owner)` 当前位置 `SetVelocity`（镰刀 chase 先例）。
  - 到家判定：`dist(主人) <= catchDist` **或** `dist <= speed×dt`（本帧必达，防高速隧穿）→ FX `patrol-catch`（主人坐标）+ `Despawn` 自己。
  - `Collision` 且 EnemyTarget 且未在回程 struck 表且 `s.Time >= hitReadyAt` → 同公式出伤 + FX `patrol-hit`，记 struck、`hitReadyAt = s.Time + hitPaceCD`。
- 主人死后引擎连坐回收，无需处理。

### 2. `character/patrol/fx/patrol.css` + `fx/patrol.js`（皮肤）

- `.ball.look-patrol`：钢蓝球；身上有 `patrol-out` mark 时（js 读 `u.marks` 折成 CSS 变量）降饱和+锚定脉冲——观众一眼看出"标识物在外、本体停机"。
- `.ball.look-patrolmark`：信号橙旋转箭标（js 注入 SVG/CSS 形），自旋速率随 `u.vx/vy` 速率加快（速度可视化=加速机制的叙事）；发光/拖尾走 Look.Glow/Trail。
- **遵守项目特效质量标准（memory）**：打击特效 `patrol-hit` 必须有错峰时间轴（瞬时炸开→延迟冲击环→延迟碎屑火花，`animation-delay` 错开），**所有带 delay 的元素初始 `opacity:0` 且动画 `forwards`**；不添加无信息量的装饰元素。

### 3. `character/patrol/fx/shot.js`（一次性特效）

`arena.registerShot` 登记四个（名字按包隔离，避开引擎保留名 hurt/heal/swap/wall*/impact/hit/faction）：`patrol-throw`（出手方向脉冲）、`patrol-hit`（错峰打击，见上；持 Pass 时引擎不发通用 hit 火花，这是唯一命中反馈）、`patrol-catch`（回收并入闪光）、`patrol-bounce`（小型墙撞火花）。

## 收尾步骤

1. `gofmt` 新文件；仓库根 `go generate ./character`（重写 all.go，再导出 `character.KindPatrol`/`KindPatrolMarker`）。
2. `go test ./...`（all_test 校验清单新鲜度）。
3. 新增 `internal/sim/patrol_test.go`（charcoal_test 范式：NewMatchSeeded+SetSlot+锁 m.mu 断言）：丢出后本体速度为 0 且标识物在场→标识物速率逐段递增→打中靶子掉血→收回后本体恢复移动→下次丢出距收回 ≥5s（种子固定，全确定性）。
4. 文档：`战斗机.md` 补「巡回」一节；`README.md` 战斗机表补一行。
5. `go run ./cmd/winrate` 更新 winrate.md（惯例收尾；矩阵较大可能要跑几分钟）。

不改动 `internal/web/*`、`main.go`；皮肤短名 `patrol`/`patrolmark` 未与 README:465 已占用清单撞车。伤害/速度常量均为包顶 const，后续按胜率微调不动结构。