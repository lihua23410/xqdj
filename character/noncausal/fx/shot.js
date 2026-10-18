/* 非因果 FX。名字和栏位跟后端 unit.FX 一致（非因果.go 的 fxName）：
   spectrum      amount=射速秒   vx=柱数          vy=脉冲数                 unitId=战斗机
   spectrum-bin  amount=柱伤害   vx=权重          vy=射速秒     slot=binIndex unitId=战斗机
   doom-at       amount=T†秒     vx=GhostHP上限   vy=射速秒     slot=槽       unitId=战斗机
   probe-hit     amount=伤害     vx=绝对时间 t                              unitId=战斗机
   lock-shot     amount=drawnAmount vx/vy=初速    x/y=枪口      slot=binIndex unitId=战斗机
   lock-armed    amount=drawnAmount                           slot=binIndex unitId=弹
   lock-block    amount=0        vx=挡住者id(墙0) vy=0墙/1单位  slot=binIndex unitId=弹
   ghost-hit     amount=drawnAmount vx=剩余GhostHP vy=弹id     slot=binIndex unitId=残影
   ghost-down    amount=最后一发 vx=敌人id        vy=GhostHP上限 slot=binIndex unitId=战斗机
   doom          amount=T†                                      slot=槽       unitId=战斗机
   不开逐发抽卡，也不做频谱扫描。大包才短闪，密了就并成一闪。 */
(function () {
  var root = typeof window !== "undefined" ? window : globalThis;

  function present() {
    return root.NoncausalPresent;
  }

  function reg(name, fn) {
    var a = typeof arena !== "undefined" ? arena : root.arena;
    if (a && typeof a.registerShot === "function") a.registerShot(name, fn);
  }

  function clock() {
    return typeof performance !== "undefined" ? performance.now() : 0;
  }

  function onFeed(fx) {
    var P = present();
    if (P && fx) P.note(fx, clock());
  }

  function onFire(fx, ctx) {
    var P = present();
    if (!P || !fx) return;
    var res = P.note(fx, clock());
    if (!res || res.flash === "none") return;
    var a = typeof arena !== "undefined" ? arena : root.arena;
    if (!a || typeof a.spawnFx !== "function") return;
    var existing = typeof document !== "undefined" ? document.querySelector(".fx.nc-packet") : null;
    if (res.flash === "merge" && existing) {
      existing.classList.add("is-merged");
      existing.style.animation = "none";
      void existing.offsetWidth;
      existing.style.animation = "";
      return;
    }
    var x = ctx && ctx.x != null ? ctx.x : 0;
    var y = ctx && ctx.y != null ? ctx.y : 0;
    var el = a.spawnFx("nc-packet", x, y, ctx && ctx.kind);
    if (el && res.flash === "merge") el.classList.add("is-merged");
  }

  function onBlock(fx, ctx) {
    var P = present();
    if (P && fx) P.note(fx, clock());
    var a = typeof arena !== "undefined" ? arena : root.arena;
    if (!a || typeof a.spawnFx !== "function") return;
    var x = ctx && ctx.x != null ? ctx.x : 0;
    var y = ctx && ctx.y != null ? ctx.y : 0;
    var el = a.spawnFx("nc-sever", x, y, ctx && ctx.kind);
    if (el) el.textContent = "因果被截";
  }

  function onGhost(fx, ctx) {
    var P = present();
    if (P && fx) P.note(fx, clock());
    var a = typeof arena !== "undefined" ? arena : root.arena;
    if (!a || typeof a.spawnFx !== "function") return;
    var x = ctx && ctx.x != null ? ctx.x : 0;
    var y = ctx && ctx.y != null ? ctx.y : 0;
    a.spawnFx("nc-tear", x, y, ctx && ctx.kind);
  }

  function onDown(fx, ctx) {
    var P = present();
    if (P && fx) P.note(fx, clock());
    var a = typeof arena !== "undefined" ? arena : root.arena;
    if (!a || typeof a.spawnFx !== "function") return;
    a.spawnFx("nc-down", ctx && ctx.x, ctx && ctx.y, ctx && ctx.kind);
  }

  function onDoom(fx, ctx) {
    var P = present();
    if (P && fx) P.note(fx, clock());
    var a = typeof arena !== "undefined" ? arena : root.arena;
    if (!a || typeof a.spawnFx !== "function") return;
    var el = a.spawnFx("nc-doom", ctx && ctx.x, ctx && ctx.y, ctx && ctx.kind);
    if (el) el.textContent = "T†";
  }

  reg("spectrum", onFeed);
  reg("spectrum-bin", onFeed);
  reg("doom-at", onFeed);
  reg("probe-hit", onFeed);
  reg("lock-shot", onFire);
  reg("lock-armed", onFeed);
  reg("lock-block", onBlock);
  reg("ghost-hit", onGhost);
  reg("ghost-down", onDown);
  reg("doom", onDoom);
})();
