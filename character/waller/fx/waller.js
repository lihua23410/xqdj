// 球体、导引与施放全部沿用主题色；动效只读对局时间，不修改墙。
(function () {
  const kind = "筑墙者";
  const duration = 0.3;
  const shell = `<svg class="waller-shell" viewBox="0 0 100 100" aria-hidden="true">
    <circle class="waller-rim" cx="50" cy="50" r="46"/>
    <g class="waller-dial">
      <circle cx="50" cy="50" r="37"/>
      <path d="M50 10v9m40 31h-9M50 90v-9M10 50h9"/>
    </g>
    <g class="waller-core">
      <path class="waller-frame" d="M32 40v-8h8m20 0h8v8m0 20v8h-8m-20 0h-8v-8"/>
      <path class="waller-joint" d="m50 36 14 14-14 14-14-14Z"/>
      <path class="waller-cross" d="M43 50h14m-7-7v14"/>
    </g>
  </svg>`.replace(/>\s+</g, "><");

  function mount(el) {
    if (!el.querySelector(":scope > .waller-shell")) el.insertAdjacentHTML("beforeend", shell);
  }

  function pulse(time, at) {
    const age = time - at;
    return age >= 0 && age < duration ? 1 - age / duration : 0;
  }

  window.lookFX = window.lookFX || {};
  window.lookFX.waller = {
    guide(u, ctx) {
      const node = ctx.ensureGuide(`waller-state-${u.id}`, "waller-state");
      const time = ctx.time || 0;
      if (!node._waller || time < node._waller.time) {
        node._waller = { time, castAt: -Infinity };
        node.innerHTML = `<svg viewBox="-50 -50 100 100" aria-hidden="true">${
          Array.from({ length: 6 }, () => '<rect x="-1.8" y="-1.8" width="3.6" height="3.6"/>').join("")
        }</svg>`;
      }
      const data = node._waller;
      data.time = time;
      for (const fx of ctx.effects || []) {
        if (fx.name !== "waller-build" || fx.kind !== kind || fx.unitId !== u.id) continue;
        // 源时间去重：暂停重复快照或延迟到达的事件都不重播。
        if (Number.isFinite(fx.amount) && fx.amount <= time + 1e-6 && fx.amount > data.castAt) {
          data.castAt = fx.amount;
          data.x = fx.x;
          data.y = fx.y;
        }
      }
      const strength = pulse(time, data.castAt);
      node.style.display = strength > 0 ? "block" : "none";
      if (strength > 0) {
        const phase = 1 - strength;
        const [x, y] = arena.screenPos(data.x, data.y, ctx.scale, ctx.cx, ctx.cy);
        const size = u.radius * 5.2 * ctx.scale;
        Object.assign(node.style, {
          left: `${x}px`, top: `${y}px`, width: `${size}px`, height: `${size}px`,
          opacity: `${Math.min(1, strength * 1.7)}`,
        });
        node.style.setProperty("--kind", arena.kindColor(kind));
        node.querySelectorAll("rect").forEach((piece, i) => {
          const angle = (i * 60 + 15) * Math.PI / 180;
          const reach = 22 + phase * 23;
          piece.setAttribute("transform", `translate(${Math.cos(angle) * reach} ${Math.sin(angle) * reach}) rotate(${i * 30 + phase * 55})`);
        });
      }
      ctx.seenGuides.add(node.id);
      for (const [id, cls] of [[`guide-link-${u.id}`, "waller-link"], [`guide-wall-${u.id}`, "waller-hint"]]) {
        // 只装饰公共层本帧已经确认的导引，不复制索敌或落点算法。
        if (!ctx.seenGuides.has(id)) continue;
        const line = document.getElementById(id);
        line.classList.add(cls);
        line.style.setProperty("--waller-pulse", strength.toFixed(4));
      }
    },
    tick(el, u, ctx = {}) {
      mount(el);
      const time = ctx.time || 0;
      const data = document.getElementById(`waller-state-${u.id}`)?._waller;
      const strength = data ? pulse(time, data.castAt) : 0;
      el.style.setProperty("--waller-turn", `${(time * 16) % 360}deg`);
      el.style.setProperty("--waller-pulse", strength.toFixed(4));
    },
    unmount(el) {
      el.querySelector(":scope > .waller-shell")?.remove();
      el.style.removeProperty("--waller-turn");
      el.style.removeProperty("--waller-pulse");
    },
  };

  // 实际绘制由上面的 guide 按快照驱动，避免通用层生成多余的一次性节点。
  arena.registerShot("waller-build", () => {});
})();
