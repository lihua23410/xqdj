// 小幺哥前端：礼物包装球、会摇晃的礼物盒，以及开盒后蹦出随机战斗机的彩带爆发。
(function () {
  const hooks = window.lookFX = window.lookFX || {};
  const states = window.xiaoyaogeState = window.xiaoyaogeState || {};
  const number = n => Number.isFinite(Number(n)) ? Number(n) : 0;
  const clamp = (n, lo, hi) => Math.max(lo, Math.min(hi, n));

  function ensure(el, cls, tag = "span") {
    let node = el.querySelector(`:scope > .${cls}`);
    if (!node) {
      node = document.createElement(tag);
      node.className = cls;
      el.appendChild(node);
    }
    return node;
  }

  function bodyArt() {
    return `
      <span class="xiaoyaoge-wrap"></span>
      <i class="xiaoyaoge-ribbon vertical"></i>
      <i class="xiaoyaoge-ribbon horizontal"></i>
      <span class="xiaoyaoge-bow"><i class="left"></i><i class="right"></i><i class="knot"></i></span>
      <span class="xiaoyaoge-face"><i class="eye left"></i><i class="eye right"></i><i class="mouth"></i></span>
      <span class="xiaoyaoge-cheek left"></span>
      <span class="xiaoyaoge-cheek right"></span>`;
  }

  function giftArt() {
    return `
      <span class="gift-box">
        <i class="gift-lid"></i>
        <i class="gift-ribbon-v"></i>
        <i class="gift-ribbon-h"></i>
        <i class="gift-bow left"></i>
        <i class="gift-bow right"></i>
        <i class="gift-spark s1"></i>
        <i class="gift-spark s2"></i>
        <i class="gift-spark s3"></i>
      </span>`;
  }

  hooks.xiaoyaoge = {
    guide(u, ctx) {
      const time = number(ctx.time);
      const events = (ctx.effects || []).filter(fx => fx.unitId === u.id && (fx.name === "xiaoyaoge-idle" || fx.name === "gift-throw"));
      let state = states[u.id];
      if (!state || time < state.time) state = states[u.id] = { time, throwAt: -Infinity, idleAt: -Infinity };
      for (const fx of events) {
        if (fx.name === "gift-throw") state.throwAt = time;
        if (fx.name === "xiaoyaoge-idle") state.idleAt = time;
      }
      state.time = time;
    },
    tick(el, u, ctx = {}) {
      if (!el?.classList.contains("look-xiaoyaoge")) return;
      const art = ensure(el, "xiaoyaoge-art", "div");
      if (!art.firstChild) art.innerHTML = bodyArt();
      const time = number(ctx.time);
      const state = states[u.id] || {};
      const throwAge = time - number(state.throwAt);
      const kick = throwAge >= 0 && throwAge < .32 ? Math.sin((throwAge / .32) * Math.PI) : 0;
      const breath = .5 + .5 * Math.sin(time * 2.8 + u.id * .23);
      el.style.setProperty("--xiaoyaoge-breath", breath.toFixed(4));
      el.style.setProperty("--xiaoyaoge-kick", kick.toFixed(4));
      el.classList.toggle("xiaoyaoge-throw", kick > 0);
      art.style.transform = `rotate(${(-kick * 12).toFixed(2)}deg) scale(${(1 + kick * .06).toFixed(3)})`;
    },
    unmount(el) {
      el?.querySelector(":scope > .xiaoyaoge-art")?.remove();
      el?.classList.remove("xiaoyaoge-throw");
      for (const p of ["--xiaoyaoge-breath", "--xiaoyaoge-kick"]) el?.style.removeProperty(p);
    },
  };

  hooks["xiaoyaoge-gift"] = {
    tick(el, u, ctx = {}) {
      if (!el) return;
      const art = ensure(el, "xiaoyaoge-gift-art", "span");
      if (!art.firstChild) art.innerHTML = giftArt();
      const angle = Math.atan2(-number(u.vy), number(u.vx));
      const time = number(ctx.time);
      el.style.setProperty("--gift-angle", `${angle}rad`);
      el.style.setProperty("--gift-spin", `${(time * 230 + u.id * 17) % 360}deg`);
      el.style.setProperty("--gift-bob", Math.sin(time * 10 + u.id).toFixed(4));
    },
    unmount(el) {
      el?.querySelector(":scope > .xiaoyaoge-gift-art")?.remove();
      for (const p of ["--gift-angle", "--gift-spin", "--gift-bob"]) el?.style.removeProperty(p);
    },
  };

  function spawnEffect(name, fx, ctx, size) {
    if (!arena || typeof arena.spawnFx !== "function") return null;
    const node = arena.spawnFx(name, ctx.x, ctx.y, fx.kind);
    if (!node) return null;
    node.style.setProperty("--xiaoyaoge-scale", String(number(ctx.scale) || 1));
    node.style.setProperty("--xiaoyaoge-size", `${size}px`);
    return node;
  }

  function throwEffect(fx, ctx) {
    const node = spawnEffect("fx-gift-throw", fx, ctx, 68);
    if (!node) return;
    node.style.setProperty("--throw-angle", `${Math.atan2(-number(fx.vy), number(fx.vx))}rad`);
    for (let i = 0; i < 4; i++) {
      const spark = document.createElement("i");
      spark.style.setProperty("--a", `${-34 + i * 22}deg`);
      spark.style.setProperty("--d", `${18 + i * 7}px`);
      node.appendChild(spark);
    }
  }

  function openEffect(fx, ctx) {
    const node = spawnEffect("fx-gift-open", fx, ctx, 120);
    if (!node) return;
    const data = fx.data && typeof fx.data === "object" ? fx.data : {};
    const art = document.createElement("span");
    art.className = "gift-open-art";
    const flash = document.createElement("i");
    flash.className = "gift-open-flash";
    const lid = document.createElement("i");
    lid.className = "gift-open-lid";
    const base = document.createElement("i");
    base.className = "gift-open-base";
    const label = document.createElement("span");
    label.className = "gift-open-label";
    label.textContent = String(data.helper || "神秘帮手");
    art.append(flash, lid, base, label);
    const colors = ["#ffd85a", "#ff6f7d", "#58e0cf", "#fff3be", "#ff9e4a"];
    for (let i = 0; i < 22; i++) {
      const confetti = document.createElement("i");
      confetti.className = "gift-confetti";
      confetti.style.setProperty("--a", `${(360 / 22) * i + Math.random() * 12}deg`);
      confetti.style.setProperty("--d", `${30 + Math.random() * 46}px`);
      confetti.style.setProperty("--r", `${180 + Math.random() * 420}deg`);
      confetti.style.setProperty("--delay", `${Math.random() * .08}s`);
      confetti.style.setProperty("--c", colors[i % colors.length]);
      art.appendChild(confetti);
    }
    node.appendChild(art);
    setTimeout(() => node.remove(), 1150);
  }

  arena.registerShot("xiaoyaoge-idle", () => {});
  arena.registerShot("gift-throw", throwEffect);
  arena.registerShot("gift-open", openEffect);
})();
