// 勇者的前端皮肤：手里握着的村好剑、等级/经验/同伴徽标，外加一次性特效。
// 等级和经验不是标记（标记只能画「图标+层数」），走 unit.FX 送过来自己画。
window.lookFX = window.lookFX || {};
window.heroState = window.heroState || {}; // unitId -> {level, exp, need, party}

(function () {
  function st(id) {
    let s = window.heroState[id];
    if (!s) {
      s = { level: 1, exp: 0, need: 2, party: {} };
      window.heroState[id] = s;
    }
    return s;
  }

  function baseOf(kind) {
    return (window.arena && arena.lookOf && arena.lookOf(kind).base) || "/ball/勇者";
  }

  // 面朝角：和盾斧武器图同一个口径——atan2(vx, vy)，因为图默认朝上。
  function facingDeg(u, st) {
    const vx = Number(u && u.vx) || 0;
    const vy = Number(u && u.vy) || 0;
    if (Math.hypot(vx, vy) < 1e-6) {
      return Number.isFinite(st.face) ? st.face : 0;
    }
    const deg = Math.atan2(vx, vy) * (180 / Math.PI);
    st.face = deg;
    return deg;
  }

  function ensure(el, cls) {
    let n = el.querySelector(`:scope > .${cls}`);
    if (!n) {
      n = document.createElement("div");
      n.className = cls;
      el.appendChild(n);
    }
    return n;
  }

  window.lookFX.hero = {
    unmount(el) {
      el?.querySelector(":scope > .hero-art")?.remove();
      el?.querySelector(":scope > .hero-badge")?.remove();
    },
    tick(el, u) {
      if (!el || !el.classList.contains("look-hero")) return;
      const s = window.heroState[u.id] || { level: 1, exp: 0, need: 2, party: {} };

      // 手里的剑：捡到才有，绕球心转向面朝方向。
      let art = el.querySelector(":scope > .hero-art");
      if (s.party.sword) {
        if (!art) art = ensure(el, "hero-art");
        let img = art.querySelector(":scope > .hero-weapon");
        if (!img) {
          img = document.createElement("img");
          img.className = "hero-weapon";
          img.alt = "";
          img.src = `${baseOf(u.kind)}/fx/sword.svg`;
          art.appendChild(img);
        }
        art.style.transform = `rotate(${facingDeg(u, s)}deg)`;
      } else if (art) {
        art.remove();
      }

      let b = el.querySelector(":scope > .hero-badge");
      if (!b) {
        b = document.createElement("div");
        b.className = "hero-badge";
        el.appendChild(b);
      }
      const dots = ["warrior", "mage", "saint"]
        .map((k) => `<i class="${k}${s.party[k] ? " on" : ""}"></i>`)
        .join("");
      b.innerHTML = `<b>Lv${s.level}</b><span>${s.exp}/${s.need}</span><em>${dots}</em>`;
    },
  };

  arena.registerShot("hero-lv", (fx) => {
    const s = st(fx.unitId);
    s.level = Math.max(1, Math.round(fx.amount || 1));
    s.exp = Math.round(fx.vx || 0);
    s.need = Math.max(1, Math.round(fx.vy || 2));
  });

  // amount: 1 村好剑 / 2 战士 / 3 法师 / 4 圣女
  const PARTY = { 1: "sword", 2: "warrior", 3: "mage", 4: "saint" };
  arena.registerShot("relic", (fx) => {
    const name = PARTY[Math.round(fx.amount || 0)];
    if (name) st(fx.unitId).party[name] = true;
    arena.spawnFx("fx-flash", fx.x, fx.y, fx.kind);
    arena.spawnFx("fx-ring", fx.x, fx.y, fx.kind);
    arena.burst(fx.x, fx.y, fx.kind, 12);
  });

  arena.registerShot("sword-hit", (fx, ctx) => {
    arena.spawnFx("fx-flash", ctx.x, ctx.y, fx.kind);
    arena.burst(ctx.x, ctx.y, fx.kind, 6);
  });

  arena.registerShot("levelup", (fx, ctx) => {
    arena.spawnFx("fx-shock", ctx.x, ctx.y, fx.kind);
    arena.spawnFx("fx-ring", ctx.x, ctx.y, fx.kind);
    arena.burst(ctx.x, ctx.y, fx.kind, 18);
  });

  arena.registerShot("shield-up", (fx, ctx) => {
    arena.spawnFx("fx-ring", ctx.x, ctx.y, fx.kind, { "--ring-size": "58px" });
  });

  arena.registerShot("shield", (fx, ctx) => {
    arena.spawnFx("fx-flash", ctx.x, ctx.y, fx.kind);
    arena.spawnFx("fx-shock", ctx.x, ctx.y, fx.kind);
  });

  arena.registerShot("heal-saint", (fx, ctx) => {
    const n = Math.round(fx.amount || 0);
    const el = arena.spawnFx("fx-dmg heal", ctx.x, ctx.y - 12, fx.kind, {
      "--dmg-size": "18px",
      "--dmg-dur": "0.9s",
      "--dmg-rise": "-48px",
    });
    if (el) el.textContent = `+${n}`;
  });
})();
