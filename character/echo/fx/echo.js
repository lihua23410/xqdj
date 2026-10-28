// 回声的声呐演出。
//
// 碎片每碰到敌人一次，在它身上补一格扫描；10 格满 = 锁定，碎片才开始掉血。
// 扫描窗口由引擎的 mark.until 管：10 秒到点 mark 自己消失，环也跟着收。
//
// 扫描环挂在回声自己的 lookFX 上，每帧读 ctx.units 里敌人的 marks 来画；
// 这样不用改 app.js，也能在回声死后靠 until 定时器把环收掉。
window.lookFX = window.lookFX || {};

(function () {
  const A = window.arena;
  if (!A) return;

  const rings = new Map(); // "owner:target" -> { el, owner, target, timer }
  let host = null;

  function ensureHost(root) {
    if (!root) return null;
    if (!host || !host.isConnected || host.parentElement !== root) {
      host = document.createElement("div");
      host.className = "fx echo-scans";
      root.appendChild(host);
    }
    return host;
  }

  function keyOf(owner, target) {
    return owner + ":" + target;
  }

  function drop(key) {
    const ring = rings.get(key);
    if (!ring) return;
    if (ring.timer) clearTimeout(ring.timer);
    ring.el.remove();
    rings.delete(key);
  }

  function makeRing(key, owner, target) {
    const el = document.createElement("div");
    el.className = "echo-scan";
    el.innerHTML = '<i class="echo-scan-fill"></i><i class="echo-scan-dash"></i>';
    host.appendChild(el);
    const ring = { el, owner, target, timer: 0, zeroAt: 0, prev: 0, lastAt: 0 };
    rings.set(key, ring);
    return ring;
  }

  function place(ring, o, ctx) {
    const p = A.screenPos(o.x, o.y, ctx.scale, ctx.cx, ctx.cy);
    const size = Math.max(44, (o.radius || 18) * ctx.scale * 2 + 24);
    ring.el.style.width = size + "px";
    ring.el.style.height = size + "px";
    ring.el.style.left = p[0] + "px";
    ring.el.style.top = p[1] + "px";
  }

  // 每一层独立活 lifetime 秒；用最新一次命中 + lifetime 作为 mark 归零的兜底时间，
  // 回声死后定时器也能把环收掉。
  function armTimer(ring, key, mark, ctx) {
    if (!(mark.lifetime > 0)) return;
    const zero = (ring.lastAt || ctx.time) + mark.lifetime;
    if (ring.zeroAt === zero) return;
    ring.zeroAt = zero;
    if (ring.timer) clearTimeout(ring.timer);
    const ms = Math.max(0, (zero - ctx.time) * 1000) + 80;
    const self = ring;
    ring.timer = setTimeout(function () {
      if (rings.get(key) === self) drop(key);
    }, ms);
  }

  window.lookFX.echo = {
    unmount(el) {
      const owner = el && el.dataset ? el.dataset.echoOwner : "";
      for (const [key, ring] of Array.from(rings)) {
        if (!owner || String(ring.owner) === owner) drop(key);
      }
      if (!rings.size && host) {
        host.remove();
        host = null;
      }
      if (el) el.classList.remove("echo-locked");
    },

    tick(el, u, ctx) {
      if (!el || !el.classList.contains("look-echo")) return;
      el.dataset.echoOwner = String(u.id);
      const root = ensureHost(ctx && ctx.fxRoot);
      if (!root) return;

      const kind = "回声计数" + u.id;
      const seen = new Set();
      let locked = false;

      for (const o of (ctx && ctx.units) || []) {
        if (o.slot === u.slot) continue;
        const mark = (o.marks || []).find((m) => m.kind === kind);
        if (!mark || mark.stacks <= 0) continue;

        const key = keyOf(u.id, o.id);
        seen.add(key);
        let ring = rings.get(key);
        if (!ring) ring = makeRing(key, u.id, o.id);
        place(ring, o, ctx);

        const full = mark.stacks >= 10;
        locked = locked || full;
        if (mark.stacks > ring.prev) ring.lastAt = ctx.time;
        ring.prev = mark.stacks;
        ring.el.style.setProperty(
          "--scan",
          ((Math.min(10, mark.stacks) / 10) * 360).toFixed(0) + "deg",
        );
        const left =
          mark.lifetime > 0 && mark.nextDecayAt > 0
            ? Math.max(0, (mark.nextDecayAt - ctx.time) / mark.lifetime)
            : 1;
        ring.el.style.setProperty("--left", Math.min(1, left).toFixed(3));
        ring.el.classList.toggle("locked", full);
        armTimer(ring, key, mark, ctx);
      }

      for (const key of Array.from(rings.keys())) {
        const ring = rings.get(key);
        if (ring && ring.owner === u.id && !seen.has(key)) drop(key);
      }

      el.classList.toggle("echo-locked", locked);
    },
  };

  // 碎片命中：落点一圈声呐；第 10 格那一圈变白。
  A.registerShot("echo-ping", function (fx, ctx) {
    const el = A.spawnFx("fx-echo-ping", ctx.x, ctx.y, ctx.kind);
    if (el) el.classList.toggle("locked", (fx.amount || 0) >= 10);
  });

  // 星弹炸开：金色大圈。
  A.registerShot("echo-burst", function (fx, ctx) {
    A.spawnFx("fx-echo-burst", ctx.x, ctx.y, ctx.kind);
  });
})();
