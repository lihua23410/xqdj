// 泡泡球前端：半透明泡泡、封存弹珠、护盾壳和破泡释放。
(function () {
  const hooks = window.lookFX = window.lookFX || {};
  const states = window.bubbleState = window.bubbleState || {};
  const mainStates = window.bubbleMain = window.bubbleMain || {};
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

  function bubbleArt() {
    return `<span class="bubble-main-film" aria-hidden="true">
      <span class="bubble-straw"><i class="bubble-straw-stick"></i><i class="bubble-straw-tip"></i><i class="bubble-straw-air"></i></span>
      <i class="bubble-main-light"></i>
      <i class="bubble-main-ring"></i>
    </span>`;
  }

  function shellArt() {
    return `<span class="bubble-shell-art" aria-hidden="true">
      <i class="bubble-film"></i>
      <i class="bubble-film-light"></i>
      <i class="bubble-film-edge"></i>
      <span class="bubble-contents"></span>
    </span>`;
  }

  function readData(fx) {
    return fx?.data && typeof fx.data === "object" ? fx.data : { count: 0, ammo: [] };
  }

  hooks["bubble-ball"] = {
    guide(u, ctx) {
      const time = number(ctx.time);
      const events = (ctx.effects || []).filter(fx => fx.unitId === u.id && String(fx.name || "").startsWith("bubble-"));
      const key = `${time}:${JSON.stringify(events)}`;
      let data = mainStates[u.id];
      if (!data || time < data.time) data = mainStates[u.id] = { time, key: "", blowStart: -Infinity, blowDuration: 0, blowSize: 0, releaseAt: -Infinity };
      if (key !== data.key) {
        for (const fx of events) {
          if (fx.name === "bubble-blow") {
            data.blowStart = time;
            data.blowDuration = number(fx.amount);
            data.blowSize = number(fx.vx);
            data.releaseAt = -Infinity;
          }
          if (fx.name === "bubble-release") data.releaseAt = time;
        }
        data.key = key;
      }
      data.time = time;
    },
    tick(el, u, ctx = {}) {
      if (!el?.classList.contains("look-bubble-ball")) return;
      const art = ensure(el, "bubble-ball-art", "div");
      if (!art.firstChild) art.innerHTML = bubbleArt();
      const time = number(ctx.time);
      const data = mainStates[u.id] || {};
      const blowing = number(data.blowDuration) > 0 && time >= number(data.blowStart) && time <= number(data.blowStart) + number(data.blowDuration) && number(data.releaseAt) < number(data.blowStart);
      const blowProgress = blowing ? clamp((time - number(data.blowStart)) / number(data.blowDuration), 0, 1) : 0;
      const strawAngle = Math.hypot(number(u.vx), number(u.vy)) > 8 ? Math.atan2(-number(u.vy), number(u.vx)) : 0;
      el.classList.toggle("bubble-blowing", blowing);
      el.style.setProperty("--bubble-breath", (.5 + .5 * Math.sin(time * 2.6 + u.id * .31)).toFixed(4));
      el.style.setProperty("--blow-progress", blowProgress.toFixed(4));
      el.style.setProperty("--blow-duration", `${number(data.blowDuration).toFixed(2)}s`);
      el.style.setProperty("--blow-size", number(data.blowSize).toFixed(2));
      el.style.setProperty("--straw-angle", `${strawAngle}rad`);
    },
    unmount(el) {
      el?.querySelector(":scope > .bubble-ball-art")?.remove();
      el?.classList.remove("bubble-blowing");
      for (const p of ["--bubble-breath", "--blow-progress", "--blow-duration", "--blow-size", "--straw-angle"]) el?.style.removeProperty(p);
    },
  };

  hooks["bubble-shell"] = {
    guide(u, ctx) {
      const time = number(ctx.time);
      const events = (ctx.effects || []).filter(fx => fx.unitId === u.id && (fx.name === "bubble-state" || fx.name === "bubble-merge"));
      let data = states[u.id];
      if (!data || time < data.time) data = states[u.id] = { time, key: "", ammo: [], capturedAt: -Infinity };
      const key = `${time}:${JSON.stringify(events)}`;
      if (key !== data.key && events.length) {
        const incoming = readData(events[events.length - 1]);
        data.ammo = Array.isArray(incoming.ammo) ? incoming.ammo : [];
        data.capturedAt = time;
        data.key = key;
      }
      data.time = time;
    },
    tick(el, u, ctx = {}) {
      if (!el?.classList.contains("look-bubble-shell")) return;
      const art = ensure(el, "bubble-shell-art", "span");
      if (!art.firstChild) art.innerHTML = shellArt();
      const data = states[u.id] || { ammo: [] };
      const ammo = Array.isArray(data.ammo) ? data.ammo : [];
      const load = clamp(ammo.length / 12, 0, 1);
      const time = number(ctx.time);
      el.style.setProperty("--bubble-load", load.toFixed(4));
      el.style.setProperty("--bubble-hue", String(175 + ((u.id * 37) % 80)));
      el.style.setProperty("--bubble-wobble", Math.sin(time * 2.3 + u.id * .41).toFixed(4));
      el.style.setProperty("--bubble-radius", number(u.radius).toFixed(2));
      el.classList.toggle("bubble-loaded", ammo.length > 0);
      el.classList.toggle("bubble-heavy", ammo.length >= 5);
      const box = art.querySelector(".bubble-contents");
      if (!box) return;
      while (box.children.length > ammo.length) box.lastElementChild.remove();
      while (box.children.length < ammo.length) {
        const bead = document.createElement("i");
        bead.className = "bubble-bullet";
        box.appendChild(bead);
      }
      ammo.forEach((a, i) => {
        const bead = box.children[i];
        const owner = arena.kindColor(a.ownerKind || a.kind || "");
        const bullet = arena.kindColor(a.kind || "");
        const angle = i * 2.399 + (u.id % 11) * .17;
        const ring = 12 + (i % 3) * 8;
        bead.style.left = `${50 + Math.cos(angle) * ring}%`;
        bead.style.top = `${50 + Math.sin(angle) * ring}%`;
        bead.style.setProperty("--bullet-color", bullet || "#eaffff");
        bead.style.setProperty("--owner-color", owner || "#ffffff");
        bead.style.setProperty("--bead-scale", (.8 + Math.min(1.1, number(a.damage) * .08)).toFixed(2));
      });
      el.setAttribute("aria-label", ammo.length ? `泡泡内封存 ${ammo.length} 发敌方弹体` : "空泡泡");
    },
    unmount(el) {
      el?.querySelector(":scope > .bubble-shell-art")?.remove();
      el?.classList.remove("bubble-loaded", "bubble-heavy");
      for (const p of ["--bubble-load", "--bubble-hue", "--bubble-wobble", "--bubble-radius"]) el?.style.removeProperty(p);
      el?.removeAttribute("aria-label");
    },
  };

  hooks["bubble-shot"] = {
    tick(el, u, ctx = {}) {
      const art = ensure(el, "bubble-shot-art", "i");
      art.style.setProperty("--travel", `${Math.atan2(-number(u.vy), number(u.vx))}rad`);
    },
    unmount(el) {
      el?.querySelector(":scope > .bubble-shot-art")?.remove();
    },
  };

  function effect(name, fx, ctx, extra = {}) {
    if (!arena || typeof arena.spawnFx !== "function") return;
    arena.spawnFx(name, ctx.x, ctx.y, fx.kind, extra);
  }

  function popEffect(fx, ctx) {
    const d = readData(fx);
    const node = effect("bubble-pop", fx, ctx, { "--bubble-count": String(d.count || 0) });
    if (!node) return;
    const dots = document.createElement("span");
    dots.className = "bubble-pop-dots";
    (d.ammo || []).slice(0, 12).forEach((a, i) => {
      const dot = document.createElement("i");
      dot.style.setProperty("--dot-color", arena.kindColor(a.kind || "") || "#efffff");
      dot.style.setProperty("--owner-color", arena.kindColor(a.ownerKind || "") || "#ffffff");
      dot.style.setProperty("--a", `${(360 / Math.max(1, d.count)) * i + 18}deg`);
      dot.style.setProperty("--d", `${14 + (i % 4) * 8}px`);
      dots.appendChild(dot);
    });
    node.appendChild(dots);
  }

  arena.registerShot("bubble-idle", (fx, ctx) => effect("bubble-idle", fx, ctx));
  arena.registerShot("bubble-state", () => {});
  arena.registerShot("bubble-merge", (fx, ctx) => effect("bubble-merge", fx, ctx));
  arena.registerShot("bubble-blow", (fx, ctx) => effect("bubble-blow", fx, ctx, { "--blow-duration": `${number(fx.amount).toFixed(2)}s`, "--blow-size": number(fx.vx).toFixed(2) }));
  arena.registerShot("bubble-release", (fx, ctx) => effect("bubble-release", fx, ctx));
  arena.registerShot("bubble-capture", (fx, ctx) => effect("bubble-capture", fx, ctx));
  arena.registerShot("bubble-pop", popEffect);
  arena.registerShot("bubble-fade", (fx, ctx) => effect("bubble-fade", fx, ctx));
  arena.registerShot("bubble-shot-hit", (fx, ctx) => effect("bubble-shot-hit", fx, ctx));
})();
