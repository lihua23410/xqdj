// 小队造型、村好剑、队伍徽标与仪式特效。等级和收编状态由 unit.FX 同步。
window.lookFX = window.lookFX || {};
window.heroState = window.heroState || {};

(function () {
  const pack = (document.currentScript && document.currentScript.dataset.pack) || "/ball/勇者";
  const levelupVoice = arena.voice(new Audio(pack + "/fx/levelup.ogg"), {
    mode: arena.SOUND_MODE.IMMUNE,
  });
  levelupVoice.audio.preload = "auto";

  // relic 的 amount 为 1 村好剑 / 2 战士 / 3 法师 / 4 圣女。
  const PARTY = ["sword", "warrior", "mage", "saint"];
  const COMPANIONS = PARTY.slice(1);
  const ROLE_BY_KIND = {
    村好剑: "sword",
    死战士: "warrior",
    战士: "warrior",
    死法师: "mage",
    法师: "mage",
    死圣女: "saint",
    圣女: "saint",
  };

  function st(id) {
    let s = window.heroState[id];
    if (!s) {
      s = { level: 1, exp: 0, need: 2, party: {}, face: 90 };
      window.heroState[id] = s;
    }
    return s;
  }

  // 面朝角：和盾斧武器图同一个口径——atan2(vx, vy)，因为图默认朝上。
  function facingDeg(u, s) {
    const vx = Number(u.vx) || 0;
    const vy = Number(u.vy) || 0;
    if (Math.hypot(vx, vy) >= 1e-6) {
      s.face = Math.atan2(vx, vy) * (180 / Math.PI);
    }
    return s.face;
  }

  window.lookFX.hero = {
    unmount(el) {
      el.querySelector(":scope > .hero-art")?.remove();
      el.querySelector(":scope > .hero-badge")?.remove();
      el.style.removeProperty("--hero-facing");
    },
    tick(el, u) {
      const s = st(u.id);
      el.style.setProperty("--hero-facing", `${facingDeg(u, s)}deg`);

      const art = el.querySelector(":scope > .hero-art");
      if (s.party.sword && !art) {
        const holder = document.createElement("div");
        holder.className = "hero-art";
        const img = document.createElement("img");
        img.className = "hero-weapon";
        img.alt = "";
        img.src = `${pack}/fx/sword.svg`;
        holder.appendChild(img);
        el.appendChild(holder);
      } else if (!s.party.sword) {
        art?.remove();
      }

      let badge = el.querySelector(":scope > .hero-badge");
      if (!badge) {
        badge = document.createElement("div");
        badge.className = "hero-badge";
        el.appendChild(badge);
      }
      const progress = [s.level, s.exp, s.need, ...COMPANIONS.map((role) => s.party[role] ? 1 : 0)].join(":");
      if (badge.dataset.progress !== progress) {
        const icons = COMPANIONS.map((role) => `<i data-hero-role="${role}"${s.party[role] ? ' class="on"' : ""}></i>`).join("");
        badge.innerHTML = `<b>Lv${s.level}</b><span>${s.exp}/${s.need}</span><em>${icons}</em>`;
        badge.style.setProperty("--hero-exp", `${Math.min(100, Math.max(0, 100 * s.exp / s.need))}%`);
        badge.dataset.progress = progress;
      }
    },
  };

  const roleSkin = {
    tick(el, u) {
      const role = ROLE_BY_KIND[u.kind];
      if (role && el.dataset.heroRole !== role) el.dataset.heroRole = role;
    },
    unmount(el) {
      delete el.dataset.heroRole;
    },
  };
  window.lookFX["hero-friend"] = roleSkin;
  window.lookFX["hero-relic"] = roleSkin;

  window.lookFX["hero-mage-shot"] = {
    unmount(el) {
      el.style.removeProperty("--mage-shot-angle");
    },
    tick(el, u) {
      if (u.vx || u.vy) {
        el.style.setProperty("--mage-shot-angle", `${Math.atan2(-u.vy, u.vx)}rad`);
      }
    },
  };

  function sigil(ctx, kind, role, diameter) {
    const el = arena.spawnFx("hero-sigil", ctx.x, ctx.y, kind, {
      "--hero-sigil-size": `${diameter * ctx.scale}px`,
    });
    if (el && role) el.dataset.heroRole = role;
  }

  arena.registerShot("hero-lv", (fx) => {
    const s = st(fx.unitId);
    s.level = Math.max(1, Math.round(fx.amount || 1));
    s.exp = Math.round(fx.vx || 0);
    s.need = Math.max(1, Math.round(fx.vy || 2));
  });

  arena.registerShot("relic", (fx, ctx) => {
    const role = PARTY[Math.round(fx.amount || 0) - 1];
    if (role) st(fx.unitId).party[role] = true;
    sigil(ctx, fx.kind, role, 42);
  });

  arena.registerShot("sword-hit", (fx, ctx) => {
    arena.spawnFx("hero-cut", ctx.x, ctx.y, fx.kind, {
      "--hero-cut-size": `${46 * ctx.scale}px`,
    });
  });

  arena.registerShot("levelup", (fx, ctx) => {
    levelupVoice.stop();
    levelupVoice.reset();
    levelupVoice.play();
    sigil(ctx, fx.kind, null, 65);
  });

  arena.registerShot("shield-up", (fx, ctx) => {
    sigil(ctx, fx.kind, "warrior", 46);
  });

  arena.registerShot("shield", (fx, ctx) => {
    sigil(ctx, fx.kind, "warrior", 58);
  });

  arena.registerShot("mage-cast", (fx, ctx) => {
    arena.spawnFx("hero-mage-cast", ctx.x, ctx.y, fx.kind, {
      "--mage-fx-size": `${16 * ctx.scale}px`,
    });
  });

  arena.registerShot("mage-shatter", (fx, ctx) => {
    arena.spawnFx("hero-mage-shatter", ctx.x, ctx.y, fx.kind, {
      "--mage-fx-size": `${30 * ctx.scale}px`,
    });
  });

  arena.registerShot("heal-saint", (fx, ctx) => {
    sigil(ctx, fx.kind, "saint", 56);
  });
})();
