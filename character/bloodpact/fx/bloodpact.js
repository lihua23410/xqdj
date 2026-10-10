// 血契的前端皮肤：暗红球（寄生血管纹 / 蔫置灰瘪）、身前 110° 咬合扇环、
// 血线（细 → 粗 → 搏动）、烧血与断线溅血。
//
// 扇环不走引擎的 Attach 弧单位：那种弧打中就销毁，主人根本在感知里见不到它，
// CD 记不住（实测每拍都能咬）。照勇者握剑那套——出伤由本体算，图自己画。
window.lookFX = window.lookFX || {};
window.pactState = window.pactState || {}; // unitId -> {state, face}
window.pactLine = window.pactLine || {}; // ownerId -> [{x1,y1,x2,y2,depth,until}] 多条血线

(function () {
  // 蝙蝠翅膀：两片折翼绕身体扑扇（CSS 动画），球太小，就画在球皮外侧。
  const BAT = `<svg class="pact-bat-svg" viewBox="-11 -11 22 22" aria-hidden="true">
    <path class="wing wl" d="M-2 0 Q-7 -5 -10 -2 Q-6 0 -10 2 Q-6 3 -2 1 Z"/>
    <path class="wing wr" d="M2 0 Q7 -5 10 -2 Q6 0 10 2 Q6 3 2 1 Z"/>
    <circle cx="0" cy="0" r="3.4"/>
    <path d="M-1.6 -3 L-2.4 -5.2 M1.6 -3 L2.4 -5.2" class="ear"/>
  </svg>`;
  // 咬合扇环：内 18 / 外 26 / ±55°，和 Go 里的 biteArcInner/Outer/Span 对齐。
  // viewBox 按世界单位取 ±30，所以 svg 像素尺寸 = 60 × scale（1 单位 = scale px）。
  const FAN = `<svg class="pact-fan-svg" viewBox="-30 -30 60 60" aria-hidden="true">
    <path d="M -21.3 -14.9 A 26 26 0 0 1 21.3 -14.9 L 14.74 -10.33 A 18 18 0 0 0 -14.74 -10.33 Z"/>
  </svg>`;

  const STATE_CLASS = { 0: "hungry", 1: "feed", 2: "wilt", 3: "blackout" };

  function st(id) {
    let s = window.pactState[id];
    if (!s) {
      s = { state: 0, face: 0 };
      window.pactState[id] = s;
    }
    return s;
  }

  // 图默认朝上，转面朝方向：和盾斧/勇者同一个口径 atan2(vx, vy)。
  function facingDeg(u, s) {
    const vx = Number(u && u.vx) || 0;
    const vy = Number(u && u.vy) || 0;
    if (Math.hypot(vx, vy) < 1e-6) return s.face || 0;
    s.face = Math.atan2(vx, vy) * (180 / Math.PI);
    return s.face;
  }

  window.lookFX.bloodpact = {    unmount(el) {
      el?.querySelector(":scope > .pact-skin")?.remove();
    },
    tick(el, u) {
      if (!el || !el.classList.contains("look-bloodpact")) return;
      const s = st(u.id);
      const want = STATE_CLASS[s.state] || "hungry";
      for (const c of ["hungry", "feed", "wilt", "blackout"]) {
        el.classList.toggle(c, c === want);
      }

      let skin = el.querySelector(":scope > .pact-skin");
      if (!skin) {
        skin = document.createElement("div");
        skin.className = "pact-skin";
        el.appendChild(skin);
      }
    },
    // 血线是持久线段，走 guide 钩子，不是一次性特效；可同时挂多条。
    guide(u, ctx) {
      if (!ctx || !ctx.ensureGuide || !window.arena) return;
      const now = performance.now();
      let list = (window.pactLine[u.id] || []).filter((ln) => ln.until > now);
      if (!list.length) {
        delete window.pactLine[u.id];
      } else {
        window.pactLine[u.id] = list;
      }
      for (let i = 0; i < list.length; i++) {
        const ln = list[i];
        const [x1, y1] = arena.screenPos(ln.x1, ln.y1, ctx.scale, ctx.cx, ctx.cy);
        const [x2, y2] = arena.screenPos(ln.x2, ln.y2, ctx.scale, ctx.cx, ctx.cy);
        const depth = Math.max(1, Math.min(3, Math.round(ln.depth || 1)));
        const g = ctx.ensureGuide(`pact-line-${u.id}-${i}`, "pact-line");
        g.className = `guide pact-line d${depth}`;
        ctx.placeSeg(g, x1, y1, x2, y2, u.kind);
        if (ctx.seenGuides) ctx.seenGuides.add(g.id);
      }
    },
  };

  // 血线：Go 每帧按目标 ID 序逐条报两端 + 深度（1 细 / 2 粗 / 3 深红搏动）。
  // 同一条线每帧都会重报：按目标端就近合并进原记录，不累积——
  // 否则 120ms 宽限里攒下七八帧的残影，一根线会被画成一撮。
  arena.registerShot("blood-line", (fx) => {
    const now = performance.now();
    const arr = (window.pactLine[fx.unitId] || []).filter((ln) => ln.until > now);
    let hit = null;
    for (const ln of arr) {
      if (Math.hypot(ln.x2 - fx.vx, ln.y2 - fx.vy) < 40) {
        hit = ln;
        break;
      }
    }
    if (hit) {
      hit.x1 = fx.x;
      hit.y1 = fx.y;
      hit.x2 = fx.vx;
      hit.y2 = fx.vy;
      hit.depth = Math.round(fx.amount || 1);
      hit.until = now + 120;
    } else {
      arr.push({
        x1: fx.x,
        y1: fx.y,
        x2: fx.vx,
        y2: fx.vy,
        depth: Math.round(fx.amount || 1),
        until: now + 120,
      });
    }
    window.pactLine[fx.unitId] = arr;
  });

  // 状态：0 饥饿 / 1 寄生 / 2 蔫置 / 3 熔断（变了才发一次）。
  arena.registerShot("blood-state", (fx) => {
    st(fx.unitId).state = Math.round(fx.amount || 0);
  });

  // 咬合：在被咬的那位身上闪一口（弧本体发，位置 = 命中点）。
  arena.registerShot("blood-bite", (fx, ctx) => {
    arena.spawnFx("fx-flash", ctx.x, ctx.y, fx.kind);
    arena.burst(ctx.x, ctx.y, fx.kind, 6);
  });

  window.lookFX["bloodpact-bat"] = {
    unmount(el) {
      el?.querySelector(":scope > .pact-bat")?.remove();
    },
    tick(el, u) {
      if (!el || !el.classList.contains("look-bloodpact-bat")) return;
      let n = el.querySelector(":scope > .pact-bat");
      if (!n) {
        n = document.createElement("div");
        n.className = "pact-bat";
        n.innerHTML = BAT;
        el.appendChild(n);
      }
      const sp = Math.hypot(Number(u && u.vx) || 0, Number(u && u.vy) || 0);
      const deg = sp < 1e-6 ? 0 : Math.atan2(u.vx, u.vy) * (180 / Math.PI);
      n.style.transform = `rotate(${deg}deg)`;
    },
  };

  // 召蝠：在血契身上炸一圈.
  arena.registerShot("blood-bats", (fx, ctx) => {
    arena.spawnFx("fx-flash", ctx.x, ctx.y, fx.kind);
    arena.burst(ctx.x, ctx.y, fx.kind, 10);
  });

  // 蝙蝠下嘴：在被咬的那位身上闪一点。
  arena.registerShot("blood-bat-bite", (fx, ctx) => {
    arena.spawnFx("fx-flash", ctx.x, ctx.y, fx.kind);
    arena.burst(ctx.x, ctx.y, fx.kind, 3);
  });

  arena.registerShot("blood-bond", (fx, ctx) => {
    arena.spawnFx("fx-ring", ctx.x, ctx.y, fx.kind, { "--ring-size": "48px" });
  });

  // 断线：血滴四溅。
  arena.registerShot("blood-snap", (fx, ctx) => {
    arena.spawnFx("fx-shock", ctx.x, ctx.y, fx.kind);
    arena.burst(ctx.x, ctx.y, fx.kind, 16);
  });

  // 收线：被拽的那位身上炸开醒目的一圈——"被拽住了"看得清清楚楚。
  arena.registerShot("blood-reel", (fx, ctx) => {
    arena.spawnFx("fx-shock", ctx.x, ctx.y, fx.kind);
    arena.spawnFx("fx-ring", ctx.x, ctx.y, fx.kind, { "--ring-size": "64px" });
    arena.burst(ctx.x, ctx.y, fx.kind, 10);
  });

  // 饿扑：扑出去那一刻在球上闪一下。
  arena.registerShot("blood-pounce", (fx, ctx) => {
    arena.spawnFx("fx-flash", ctx.x, ctx.y, fx.kind);
    arena.burst(ctx.x, ctx.y, fx.kind, 8);
  });
})();
