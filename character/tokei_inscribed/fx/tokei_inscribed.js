(function () {
  // 抽牌、手牌、命中抽牌冷却都只是内部机制：球上不画牌槽、不飘抽牌点、也不挂冷却环。
  // 皮肤只剩「会走的表盘」——白盘、12 刻度、两根金针，跟对局时间走。
  //
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

  // 时痕每叠一层：飘一个当前层数。
  arena.registerShot("tokei-inscribed-trace", (fx, ctx) => {
    const el = arena.spawnFx("fx-tokei-inscribed-trace", ctx.x, ctx.y - 16, fx.kind);
    if (el) el.textContent = `+${Math.round(fx.amount || 1)}`;
  });

  // 长守补池：一圈白光的护罩。
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

  // —— 常驻皮肤：和时计同一张白盘，两根针改成走真实时间 ——
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
      const f = face(el);
      // 真表盘的走法：分针 1 分钟一圈（每 10 秒 60°），时针 12 分钟一圈。
      // 一场一百来秒的对局里两根针都看得见在动，又不会像秒针那样抖。
      const t = Math.max(0, (ctx && ctx.time) || 0);
      const mh = f.querySelector(".hand.minute");
      const hh = f.querySelector(".hand.hour");
      if (mh) mh.style.transform = `rotate(${((t * 6) % 360).toFixed(2)}deg)`;
      if (hh) hh.style.transform = `rotate(${((t * 0.5) % 360).toFixed(2)}deg)`;

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
    },
    unmount(el) {
      for (const cls of ["tokei-face", "inscribed-shield"]) {
        const n = el.querySelector(`:scope > .${cls}`);
        if (n) n.remove();
      }
    },
  };
})();
