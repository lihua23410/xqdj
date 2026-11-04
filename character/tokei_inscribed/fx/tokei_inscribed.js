(function () {
  window.tokeiInscribedHUD = window.tokeiInscribedHUD || {};

  const SLOTS = 5;

  // 手牌 / 抽卡进度 / 时痕层数 / 20s 冷却就绪时刻。冷却环和两根针都由 lookFX 每帧自己算，
  // 这里只在状态变化时收到一条（进度每秒变一次，所以分针也是每秒走一格）。
  arena.registerShot("tokei-inscribed-cards", (fx) => {
    if (!fx || fx.unitId == null) return;
    const packed = Math.round(fx.vx || 0) >>> 0;
    const n = Math.max(0, Math.min(SLOTS, Math.round(fx.vy || 0)));
    const cards = [];
    for (let i = 0; i < n; i++) cards.push((packed >> (i * 4)) & 0xf);
    window.tokeiInscribedHUD[fx.unitId] = {
      cards,
      fill: Math.max(0, Math.min(1, (fx.amount || 0) / 5)),
      trace: Math.max(0, Math.round(fx.y || 0)),
      cdReadyAt: fx.x || 0,
    };
  });

  // 铭刻 / 回溯：在球的位置摊开一张半透明的表盘（时计场上那张脸的半透明版），
  // 分针原地转一整圈——铭刻顺时针、回溯逆时针，盖在最上层。
  function spin(fx, ctx, dir) {
    const dial = arena.spawnFx("fx-tokei-inscribed-clock", ctx.x, ctx.y, fx.kind);
    if (!dial) return;
    dial.dataset.dir = dir;
    const scale = ctx.scale || 1;
    const size = 96 * scale;
    dial.style.width = dial.style.height = `${size}px`;
    for (let i = 0; i < 12; i++) {
      const t = document.createElement("i");
      t.className = "tick";
      t.style.transform = `rotate(${i * 30}deg)`;
      dial.appendChild(t);
    }
    const hand = document.createElement("i");
    hand.className = "hand";
    dial.appendChild(hand);
  }
  arena.registerShot("tokei-inscribed-mark", (fx, ctx) => spin(fx, ctx, "cw"));
  arena.registerShot("tokei-inscribed-rewind", (fx, ctx) => spin(fx, ctx, "ccw"));

  // 抽到牌：球上方一个小白点弹开。
  arena.registerShot("tokei-inscribed-draw", (fx, ctx) => {
    arena.spawnFx("fx-tokei-inscribed-draw", ctx.x, ctx.y - 24, fx.kind);
  });

  // 冲刺：一道带速度方向的拉丝。
  arena.registerShot("tokei-inscribed-dash", (fx, ctx) => {
    const el = arena.spawnFx("fx-tokei-inscribed-dash", ctx.x, ctx.y, fx.kind);
    if (el) {
      const ang = Math.atan2(-(fx.vy || 0), fx.vx || 1);
      el.style.setProperty("--ang", `${ang}rad`);
    }
  });

  // 命中：小环 + 火花。
  arena.registerShot("tokei-inscribed-hit", (fx, ctx) => {
    arena.spawnFx("fx-tokei-inscribed-hit", ctx.x, ctx.y, fx.kind);
  });

  // 每层时痕：飘一个层数。
  arena.registerShot("tokei-inscribed-trace", (fx, ctx) => {
    const el = arena.spawnFx("fx-tokei-inscribed-trace", ctx.x, ctx.y - 16, fx.kind);
    if (el) el.textContent = `+${Math.round(fx.amount || 1)}`;
  });

  // 每层长守：一圈白光的护罩。
  arena.registerShot("tokei-inscribed-guard", (fx, ctx) => {
    arena.spawnFx("fx-tokei-inscribed-guard", ctx.x, ctx.y, fx.kind);
  });

  // 烬时之斩：按真实半径画一圈火环。
  arena.registerShot("tokei-inscribed-ember", (fx, ctx) => {
    const r = Math.max(24, (fx.amount || 144) * (ctx.scale || 1));
    const el = arena.spawnFx("fx-tokei-inscribed-ember", ctx.x, ctx.y, fx.kind);
    if (el) el.style.setProperty("--r", `${r}px`);
  });
  arena.registerShot("tokei-inscribed-ember-pop", (fx, ctx) => {
    arena.spawnFx("fx-tokei-inscribed-pop", ctx.x, ctx.y, fx.kind);
  });

  // —— 常驻皮肤：和时计同一张白盘，两根针接上了铭刻自己的资源 ——
  function ensure(el, cls) {
    let n = el.querySelector(`:scope > .${cls}`);
    if (!n) {
      n = document.createElement("div");
      n.className = cls;
      el.appendChild(n);
    }
    return n;
  }

  // 白盘 + 12 个刻度 + 时针分针，和时计皮肤同构。
  function face(el) {
    const f = ensure(el, "tokei-face");
    if (f.dataset.built) return f;
    f.dataset.built = "1";
    for (let i = 0; i < 12; i++) {
      const t = document.createElement("i");
      t.className = "tick";
      t.style.transform = `rotate(${i * 30}deg)`;
      f.appendChild(t);
    }
    for (const cls of ["hand hour", "hand minute"]) {
      const h = document.createElement("i");
      h.className = cls;
      f.appendChild(h);
    }
    return f;
  }

  window.lookFX = window.lookFX || {};
  window.lookFX["tokei-inscribed"] = {
    tick(el, u, ctx) {
      const st = window.tokeiInscribedHUD[u.id] || { cards: [], fill: 0, trace: 0, cdReadyAt: 0 };
      const f = face(el);

      // 分针 = 抽卡进度（5 点一圈，每点 72°）；时针 = 时痕层数（每层 30°，12 层到头）。
      const minute = Math.max(0, Math.min(5, (st.fill || 0) * 5)) * 72;
      const hour = Math.min(12, st.trace || 0) * 30;
      const mh = f.querySelector(".hand.minute");
      const hh = f.querySelector(".hand.hour");
      if (mh) mh.style.transform = `rotate(${minute}deg)`;
      if (hh) hh.style.transform = `rotate(${hour}deg)`;

      // 手牌槽
      let host = el.querySelector(":scope > .inscribed-hand");
      if (!host) {
        host = document.createElement("i");
        host.className = "inscribed-hand";
        el.appendChild(host);
      }
      for (let i = 0; i < SLOTS; i++) {
        let slot = host.children[i];
        if (!slot) {
          slot = document.createElement("i");
          slot.className = "inscribed-card";
          host.appendChild(slot);
        }
        const c = st.cards[i] || 0;
        slot.dataset.c = String(c);
        slot.classList.toggle("is-on", c > 0);
        slot.classList.toggle("is-fill", c <= 0 && i === st.cards.length);
        const p = c > 0 ? 100 : i === st.cards.length ? Math.round((st.fill || 0) * 100) : 0;
        slot.style.setProperty("--p", `${p}%`);
      }

      // 长守的额外血条：引擎的 Absorb，挂在球上方，被打掉多少就短多少。
      const pool = Math.max(0, Number(u.absorb) || 0);
      let shield = el.querySelector(":scope > .inscribed-shield");
      if (pool <= 0) {
        if (shield) shield.remove();
      } else {
        if (!shield) {
          shield = document.createElement("i");
          shield.className = "inscribed-shield";
          shield.append(document.createElement("b"), document.createElement("span"));
          el.appendChild(shield);
        }
        shield.firstElementChild.style.setProperty("--fill", `${Math.min(100, pool).toFixed(1)}%`);
        shield.lastElementChild.textContent = String(Math.max(1, Math.round(pool)));
      }

      // 冷却环：就绪时整圈亮，否则按剩余比例收。
      let cd = el.querySelector(":scope > .inscribed-cd");
      if (!cd) {
        cd = document.createElement("i");
        cd.className = "inscribed-cd";
        el.appendChild(cd);
      }
      const now = (ctx && ctx.time) || 0;
      const left = (st.cdReadyAt || 0) - now;
      const armed = left <= 1e-6;
      cd.classList.toggle("is-ready", armed);
      cd.style.setProperty("--cd", armed ? "1" : String(Math.max(0, Math.min(1, 1 - left / 20))));
    },
    unmount(el) {
      for (const cls of ["tokei-face", "inscribed-hand", "inscribed-cd", "inscribed-shield"]) {
        const n = el.querySelector(`:scope > .${cls}`);
        if (n) n.remove();
      }
    },
  };
})();
