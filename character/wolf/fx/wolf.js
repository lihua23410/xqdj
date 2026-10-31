// 私有展示状态按主人保存；只读月相/撕咬事件和真实扇环，不参与战斗。
(function () {
  const hooks = window.lookFX = window.lookFX || {};
  const body = `<svg class="wolf-appendages" viewBox="0 0 100 100" aria-hidden="true">
    <g class="wolf-mane">
      <path d="M10 25-2 31 5 42-8 47 3 55-4 67 10 65 9 79 24 72 24 55Z"/>
      <path d="m90 25 12 6-7 11 13 5-11 8 7 12-14-2 1 14-15-7V55Z"/>
      <path d="m28 88 7 15 8-5 7 9 7-9 8 5 7-15Z"/>
    </g>
    <path class="wolf-ear-shell" d="M10 25 20-13 40 7Q22 12 10 25Zm80 0L80-13 60 7q18 5 30 18Z"/>
    <path class="wolf-ear-fold" d="m17 15 5-17 10 11Zm66 0L78-2 68 9Z"/>
    <path class="wolf-ear-edge" d="m13 20 7-30 15 16m52 14-7-30-15 16"/>
  </svg><span class="wolf-pelt"></span>
  <svg class="wolf-jaws" viewBox="0 0 100 100" aria-hidden="true">
    <g class="wolf-upper-jaw"><path class="wolf-canine" d="M94 28Q104 31 108 49L99 40Z"/><path class="wolf-small-fang" d="m98 38 11 8-9-2Z"/></g>
    <g class="wolf-lower-jaw"><path class="wolf-canine" d="M94 72Q104 69 108 51L99 60Z"/><path class="wolf-small-fang" d="m98 62 11-8-9 2Z"/></g>
  </svg>`;
  const state = id => document.getElementById(`wolf-state-${id}`)?._wolf;
  window.wolfVisual = { state };
  hooks.wolf = {
    guide(u, ctx) {
      const node = ctx.ensureGuide(`wolf-state-${u.id}`, "wolf-state");
      const time = ctx.time || 0;
      if (!node._wolf || time < node._wolf.time) {
        node._wolf = { time, phase: 0, phaseAt: -Infinity, raging: false, rageAt: -Infinity, biteAt: -Infinity, arcId: null, key: "" };
      }
      const data = node._wolf;
      for (const fx of ctx.effects || []) {
        if (fx.kind !== u.kind) continue;
        // 老录像没有月相归属时，只在该阵营唯一狼人时兼容。
        const legacyPhase = fx.name === "phase" && !fx.unitId && fx.slot === u.slot && ctx.units.filter(n => n.kind === u.kind && n.slot === u.slot).length === 1;
        if (fx.unitId !== u.id && !legacyPhase) continue;
        if (fx.name === "phase") {
          const key = `${time}:${fx.amount}`;
          if (data.key !== key) {
            data.key = key;
            data.phase = Math.max(0, Math.min(7, fx.amount | 0));
            data.phaseAt = time;
          }
        } else if (fx.name === "rage") {
          if (fx.amount > 0 && !data.raging) data.rageAt = time;
          data.raging = fx.amount > 0;
        }
      }
      const arc = ctx.units.find(n => n.ownerId === u.id && n.kind === "狼人撕咬弧");
      if (arc && arc.id !== data.arcId) data.biteAt = time;
      data.arcId = arc?.id ?? null;
      data.time = time;
      node.style.display = "none";
      ctx.seenGuides.add(node.id);
    },
    tick(el, u, ctx) {
      let art = el.querySelector(":scope > .wolf-body");
      if (!art) {
        art = document.createElement("span");
        art.className = "wolf-body";
        art.setAttribute("aria-hidden", "true");
        art.innerHTML = body;
        el.appendChild(art);
      }
      const data = state(u.id);
      const arc = ctx.units.find(n => n.ownerId === u.id && n.kind === "狼人撕咬弧");
      const raging = !!data?.raging || !!arc;
      const time = ctx.time || 0;
      const age = time - (data?.rageAt ?? -Infinity);
      const snap = Math.max(0, 1 - (time - (data?.biteAt ?? -Infinity)) / .18);
      const face = ctx.units.find(n => n.ownerId === u.id && n.arcSpan > 0) || u;
      const hasFace = Math.hypot(face.faceX || 0, face.faceY || 0) > 1e-9;
      const dx = hasFace ? face.faceX : (face.vx || 1);
      const dy = hasFace ? face.faceY : (face.vy || 0);
      art.querySelector(".wolf-jaws").style.transform = `rotate(${Math.atan2(-dy, dx)}rad)`;
      const pulse = raging ? 0.72 + 0.28 * Math.sin(time * 11) ** 2 : 0;
      el.classList.toggle("wolf-rage", raging);
      el.style.setProperty("--wolf-rage", pulse.toFixed(4));
      el.style.setProperty("--wolf-jaw", (raging ? 1.8 + 6 * snap : .4).toFixed(3) + "px");
      el.style.setProperty("--wolf-flash", (raging && age >= 0 && age < 0.22 ? 1 - age / 0.22 : 0).toFixed(4));
    },
    unmount(el) {
      el.querySelector(":scope > .wolf-body")?.remove();
      el.classList.remove("wolf-rage");
      for (const prop of ["--wolf-rage", "--wolf-jaw", "--wolf-flash"]) el.style.removeProperty(prop);
    },
  };
})();
