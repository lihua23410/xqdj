// 凝聚体前端：液态主体、质量读数、凝块回收与速度抵押反馈。
window.coalesceSt = window.coalesceSt || {};
window.lookFX = window.lookFX || {};

(function () {
  const hooks = window.lookFX;
  const chunkValues = {
    "凝聚体小凝块": 3,
    "凝聚体中凝块": 6,
    "凝聚体重凝块": 10,
    "凝聚体弹块": 12,
  };

  function state(id) {
    let s = window.coalesceSt[id];
    if (!s) {
      s = {
        time: -Infinity,
        compressUntil: -Infinity,
        absorbUntil: -Infinity,
        denied: new Map(),
      };
      window.coalesceSt[id] = s;
    }
    return s;
  }

  function sync(id, time) {
    const s = state(id);
    if (Number.isFinite(time) && time + 1e-6 < s.time) {
      s.denied.clear();
      s.compressUntil = -Infinity;
      s.absorbUntil = -Infinity;
    }
    if (Number.isFinite(time)) s.time = time;
    return s;
  }

  function ensure(el, cls, tag = "div") {
    let node = el.querySelector(`:scope > .${cls}`);
    if (!node) {
      node = document.createElement(tag);
      node.className = cls;
      el.appendChild(node);
    }
    return node;
  }

  function timeOf(ctx) {
    return ctx && Number.isFinite(ctx.time) ? ctx.time : 0;
  }

  function spawnEffect(ctx, cls) {
    if (!ctx || ctx.audioOnly) return null;
    const root = document.getElementById("fx");
    if (!root) return null;
    const node = document.createElement("div");
    node.className = `fx ${cls}`;
    node.style.left = `${ctx.x}px`;
    node.style.top = `${ctx.y}px`;
    root.appendChild(node);
    node.addEventListener("animationend", () => node.remove(), { once: true });
    setTimeout(() => node.remove(), 900);
    return node;
  }

  const bodyArt = `<div class="coalesce-gel">
    <span class="coalesce-sheen"></span>
    <span class="coalesce-core"></span>
    <span class="coalesce-bubble b0"></span>
    <span class="coalesce-bubble b1"></span>
    <span class="coalesce-bubble b2"></span>
    <span class="coalesce-bubble b3"></span>
    <span class="coalesce-skin-line l0"></span>
    <span class="coalesce-skin-line l1"></span>
  </div>`;

  hooks.coalesce = {
    unmount(el) {
      if (!el) return;
      el.querySelector(":scope > .coalesce-art")?.remove();
      el.classList.remove("coalesce-low", "coalesce-heavy", "coalesce-compressing", "coalesce-absorbing");
    },
    tick(el, u, ctx = {}) {
      if (!el || !el.classList.contains("look-coalesce")) return;
      const s = sync(u.id, ctx.time);
      const art = ensure(el, "coalesce-art");
      if (!art.firstElementChild) art.innerHTML = bodyArt;
      const max = Math.max(1, Number(u.maxHp) || 100);
      const hp = Math.max(0, Number(u.hp) || 0);
      const mass = Math.max(0, Math.min(1, hp / max));
      const speed = Math.hypot(Number(u.vx) || 0, Number(u.vy) || 0);
      el.style.setProperty("--coalesce-mass", mass.toFixed(4));
      el.style.setProperty("--coalesce-speed", Math.min(2.4, speed / 190).toFixed(4));
      el.classList.toggle("coalesce-low", mass < 0.34);
      el.classList.toggle("coalesce-heavy", mass > 0.72);
      el.classList.toggle("coalesce-compressing", timeOf(ctx) < s.compressUntil);
      el.classList.toggle("coalesce-absorbing", timeOf(ctx) < s.absorbUntil);
    },
    guide(u, ctx) {
      if (!u || !ctx || !ctx.ensureGuide || !window.arena) return;
      const s = sync(u.id, ctx.time);
      const time = timeOf(ctx);
      for (const [targetId, until] of s.denied) {
        if (until <= time) {
          s.denied.delete(targetId);
          continue;
        }
        const target = (ctx.units || []).find((o) => o && o.id === targetId);
        if (!target) continue;
        const [x, y] = arena.screenPos(target.x, target.y, ctx.scale, ctx.cx, ctx.cy);
        const node = ctx.ensureGuide(`coalesce-weight-${u.id}-${targetId}`, "coalesce-weight-ring");
        node.className = "guide coalesce-weight-ring";
        node.style.left = `${x}px`;
        node.style.top = `${y}px`;
        node.style.width = `${(target.radius + 14) * 2}px`;
        node.style.height = `${(target.radius + 14) * 2}px`;
        node.style.setProperty("--ang", "0rad");
        node.style.setProperty("--kind", arena.kindColor(u.kind));
        ctx.seenGuides.add(node.id);
      }
    },
  };

  hooks["coalesce-chunk"] = {
    unmount(el) {
      el?.querySelector(":scope > .coalesce-drop-art")?.remove();
      el?.querySelector(":scope > .coalesce-mass-label")?.remove();
    },
    tick(el, u) {
      if (!el) return;
      const value = chunkValues[u.kind] || 3;
      el.classList.toggle("coalesce-mass-small", value === 3);
      el.classList.toggle("coalesce-mass-medium", value === 6);
      el.classList.toggle("coalesce-mass-heavy", value === 10);
      el.classList.toggle("coalesce-mass-shot", value === 12);
      const art = ensure(el, "coalesce-drop-art");
      if (!art.firstElementChild) {
        art.innerHTML = '<span class="coalesce-drop-core"></span><i></i><i></i><i></i>';
      }
      const label = ensure(el, "coalesce-mass-label", "b");
      label.textContent = String(value);
    },
  };

  hooks["coalesce-shot"] = {
    unmount(el) {
      el?.querySelector(":scope > .coalesce-shot-art")?.remove();
    },
    tick(el, u) {
      if (!el) return;
      const art = ensure(el, "coalesce-shot-art");
      if (!art.firstElementChild) {
        art.innerHTML = '<span class="coalesce-shot-core"></span><span class="coalesce-shot-tail"></span>';
      }
      const speed = Math.hypot(Number(u.vx) || 0, Number(u.vy) || 0);
      const ang = Math.atan2(-(Number(u.vy) || 0), Number(u.vx) || 1);
      art.style.transform = `rotate(${ang}rad) scaleX(${(0.82 + Math.min(0.5, speed / 900)).toFixed(3)})`;
    },
  };

  arena.registerShot("coalesce-compress", (fx, ctx) => {
    const s = state(fx.unitId);
    s.compressUntil = timeOf(ctx) + 0.27;
    spawnEffect(ctx, "fx-coalesce-compress");
  });

  arena.registerShot("coalesce-shed", (fx, ctx) => {
    spawnEffect(ctx, "fx-coalesce-shed");
  });

  arena.registerShot("coalesce-absorb", (fx, ctx) => {
    const s = state(fx.unitId);
    s.absorbUntil = timeOf(ctx) + 0.24;
    spawnEffect(ctx, "fx-coalesce-absorb");
  });

  arena.registerShot("coalesce-deny", (fx, ctx) => {
    const s = state(fx.unitId);
    const targetId = Math.round(Number(fx.vx) || 0);
    if (targetId) s.denied.set(targetId, timeOf(ctx) + 2.8);
    spawnEffect(ctx, "fx-coalesce-deny");
    if (ctx && window.arena) arena.burst(ctx.x, ctx.y, fx.kind, 8);
  });

  arena.registerShot("coalesce-collide", (fx, ctx) => {
    const node = spawnEffect(ctx, "fx-coalesce-collide");
    if (node) node.style.setProperty("--hit", (0.3 + Math.min(1, (Number(fx.amount) || 1) / 12) * 0.7).toFixed(3));
    if (ctx && window.arena) arena.burst(ctx.x, ctx.y, fx.kind, 6);
  });

  arena.registerShot("coalesce-shot-hit", (fx, ctx) => {
    spawnEffect(ctx, "fx-coalesce-shot-hit");
    if (ctx && window.arena) arena.burst(ctx.x, ctx.y, fx.kind, 12);
  });

  arena.registerShot("coalesce-return", (fx, ctx) => {
    spawnEffect(ctx, "fx-coalesce-return");
  });
})();
