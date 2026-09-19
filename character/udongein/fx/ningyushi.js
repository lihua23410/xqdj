window.lookFX = window.lookFX || {};
window.ningyushiHUD = window.ningyushiHUD || {};

const NINGYUSHI_POSE = {
  0: { path: "素材/objectAa/objectAa", n: 4, fps: 6 },
  1: { path: "素材/objectAb/objectAb", n: 2, fps: 10 },
  2: { path: "素材/objectAc/objectAc", n: 4, fps: 10 },
  3: { path: "素材/objectAd/objectAd", n: 8, fps: 12 },
  4: { path: "素材/objectAe/objectAe", n: 6, fps: 10 },
  5: { path: "素材/objectAh/objectAh", n: 2, fps: 8 },
  6: { path: "素材/objectAi/objectAi", n: 2, fps: 8 },
};

function ningyushiBase(kind) {
  const look = window.arena && typeof arena.lookOf === "function" ? arena.lookOf(kind || "人偶使") : {};
  return look.base || "/ball/人偶使";
}

function ningyushiFrame(pose, now) {
  const p = NINGYUSHI_POSE[pose] || NINGYUSHI_POSE[0];
  const i = Math.floor(((now || 0) / 1000) * p.fps) % p.n;
  return `${ningyushiBase()}/${p.path}${String(i).padStart(3, "0")}.png`;
}

function ningyushiFace(u, ctx) {
  if (u && Math.hypot(u.vx || 0, u.vy || 0) > 8) {
    const f = (u.vx || 0) >= 0 ? -1 : 1;
    if (u.role !== "fighter") console.log("[face] by velocity", u.kind, "vx=", u.vx, "face=", f);
    return f;
  }
  const units = (ctx && ctx.units) || [];
  const foe = units.find((o) => o && o.role === "fighter" && o.slot !== u.slot);
  if (foe) {
    const f = foe.x >= u.x ? -1 : 1;
    if (u.role !== "fighter") console.log("[face] by foe", u.kind, "foe.x=", foe.x, "u.x=", u.x, "face=", f);
    return f;
  }
  // 找不到敌方 fighter 时，尝试朝向主人方向（人偶/炸弹）
  const owner = units.find((o) => o && o.role === "fighter" && o.slot === u.slot);
  if (owner) {
    const f = owner.x >= u.x ? -1 : 1;
    if (u.role !== "fighter") console.log("[face] by owner", u.kind, "owner.x=", owner.x, "u.x=", u.x, "face=", f);
    return f;
  }
  if (u.role !== "fighter") console.log("[face] fallback", u.kind, "face=1");
  return 1;
}

function ningyushiSpr(el, pose, face, scale) {
  let spr = el.querySelector(":scope > .ningyushi-spr");
  if (!spr) {
    spr = document.createElement("i");
    spr.className = "ningyushi-spr";
    el.appendChild(spr);
  }
  const h = Math.max(8, 20 * (scale || 1));
  spr.style.backgroundImage = `url("${ningyushiFrame(pose, performance.now())}")`;
  spr.style.setProperty("--spr-h", `${h}px`);
  spr.style.setProperty("--face", String(face || 1));
}

window.lookFX.ningyushi = {
  unmount(el) {
    if (!el) return;
    el.style.removeProperty("--ningyushi-energy");
    el.removeAttribute("data-ningyushi-drained");
    el.querySelector(":scope > .ningyushi-cards")?.remove();
    el.querySelector(":scope > .ningyushi-fan")?.remove();
  },
  tick(el, u, ctx) {
    if (!el) return;
    const st = window.ningyushiHUD[u.id] || { energy: 5, cap: 5, cards: [], fill: 0 };
    const fan = st.fan;
    let wedge = el.querySelector(":scope > .ningyushi-fan");
    if (!fan || !(fan.span > 0)) {
      wedge?.remove();
    } else {
      if (!wedge) {
        wedge = document.createElement("i");
        wedge.className = "ningyushi-fan";
        el.appendChild(wedge);
      }
      const scale = (ctx && ctx.scale) || 1;
      const span = fan.span;
      const reach = (fan.reach || 36) * scale;
      const ang = Math.atan2(-(fan.vy || 0), fan.vx || 1);
      wedge.style.setProperty("--ox", `${(fan.x - u.x) * scale}px`);
      wedge.style.setProperty("--oy", `${-(fan.y - u.y) * scale}px`);
      wedge.style.setProperty("--reach", `${reach}px`);
      wedge.style.setProperty("--span", `${span}rad`);
      wedge.style.setProperty("--from", `${Math.PI / 2 - span / 2}rad`);
      wedge.style.setProperty("--ang", `${ang}rad`);
    }
    const e = Math.max(0, Math.min(1, (st.energy || 0) / 5));
    el.style.setProperty("--ningyushi-energy", String(e));
    el.dataset.ningyushiDrained = (st.cap || 5) < 5 ? "1" : "0";
    let host = el.querySelector(":scope > .ningyushi-cards");
    if (!host) {
      host = document.createElement("i");
      host.className = "ningyushi-cards";
      el.appendChild(host);
    }
    const cards = Array.isArray(st.cards) ? st.cards : [];
    const fill = Math.max(0, Math.min(1, st.fill || 0));
    for (let i = 0; i < 5; i++) {
      let slot = host.children[i];
      if (!slot) {
        slot = document.createElement("i");
        slot.className = "ningyushi-card";
        host.appendChild(slot);
      }
      const sk = cards[i] || 0;
      slot.dataset.sk = String(sk);
      slot.classList.toggle("is-on", sk > 0);
      slot.classList.toggle("is-fill", sk <= 0 && i === cards.length);
      slot.style.setProperty("--p", sk > 0 ? "100%" : i === cards.length ? `${Math.round(fill * 100)}%` : "0%");
    }
  },
  guide(u, ctx) {
    if (!u || u.role !== "fighter" || !ctx.ensureGuide || !ctx.placeSeg || !window.arena) return;
    const st = window.ningyushiHUD[u.id] || {};
    const dolls = (ctx.units || []).filter((o) => o && o.kind === "人偶使偶" && o.ownerId === u.id);
    for (const d of dolls) {
      const [x1, y1] = arena.screenPos(u.x, u.y, ctx.scale, ctx.cx, ctx.cy);
      const [x2, y2] = arena.screenPos(d.x, d.y, ctx.scale, ctx.cx, ctx.cy);
      const g = ctx.ensureGuide(`guide-ningyushi-${u.id}-${d.id}`, "ningyushi-line");
      ctx.placeSeg(g, x1, y1, x2, y2, u.kind);
      g.style.setProperty("--kind", "#fff");
      if (ctx.seenGuides) ctx.seenGuides.add(g.id);
    }
    const segs = Array.isArray(st.laser) ? st.laser : [];
    segs.forEach((seg, i) => {
      if (!seg) return;
      const [x1, y1] = arena.screenPos(seg.x1, seg.y1, ctx.scale, ctx.cx, ctx.cy);
      const [x2, y2] = arena.screenPos(seg.x2, seg.y2, ctx.scale, ctx.cx, ctx.cy);
      const g = ctx.ensureGuide(`guide-ningyushi-laser-${u.id}-${i}`, "ningyushi-laser");
      ctx.placeSeg(g, x1, y1, x2, y2, u.kind);
      if (ctx.seenGuides) ctx.seenGuides.add(g.id);
    });
  },
};

window.lookFX["ningyushi-doll"] = {
  unmount(el) {
    el?.querySelector(":scope > .ningyushi-ellipse")?.remove();
    el?.querySelector(":scope > .ningyushi-spr")?.remove();
  },
  tick(el, u, ctx) {
    if (!el) return;
    const st = window.ningyushiHUD[u.id] || {};
    const scale = (ctx && ctx.scale) || 1;
    ningyushiSpr(el, st.pose || 0, ningyushiFace(u, ctx), scale);
    const on = st.range;
    let ring = el.querySelector(":scope > .ningyushi-ellipse");
    if (!on) {
      ring?.remove();
      return;
    }
    if (!ring) {
      ring = document.createElement("i");
      ring.className = "ningyushi-ellipse";
      el.appendChild(ring);
    }
    ring.style.setProperty("--scale", String(scale));
    ring.style.setProperty("--ang", "0rad");
  },
};

window.lookFX["ningyushi-bomb"] = {
  unmount(el) {
    el?.querySelector(":scope > .ningyushi-spr")?.remove();
  },
  tick(el, u, ctx) {
    if (!el) return;
    const st = window.ningyushiHUD[u.id] || {};
    const pose = Number.isFinite(st.pose) ? st.pose : 6;
    ningyushiSpr(el, pose, ningyushiFace(u, ctx), (ctx && ctx.scale) || 1);
  },
};

window.lookFX["ningyushi-shot"] = { unmount() {}, tick() {} };

window.lookFX["ningyushi-beam"] = {
  unmount(el) {
    el?.querySelector(":scope > .ningyushi-ray")?.remove();
  },
  tick(el, u, ctx) {
    if (!el) return;
    let ray = el.querySelector(":scope > .ningyushi-ray");
    if (!ray) {
      ray = document.createElement("div");
      ray.className = "ningyushi-ray";
      el.appendChild(ray);
    }
    const scale = (ctx && ctx.scale) || 1;
    const dist = Math.hypot(u.vx || 0, u.vy || 0);
    const ang = Math.atan2(-(u.vy || 0), u.vx || 1);
    const len = (dist > 8 ? dist : 560) * scale;
    ray.style.setProperty("--beam-ang", `${ang}rad`);
    ray.style.setProperty("--beam-len", `${len}px`);
  },
};
