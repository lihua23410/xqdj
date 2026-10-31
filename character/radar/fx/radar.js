// 所有装饰都留在球内；激光只读后端已经按墙截断的方向和长度。
(function () {
  const glass = `<span class="radar-glass" aria-hidden="true">
    <svg viewBox="0 0 100 100">
      <circle class="radar-inner-rim" cx="50" cy="50" r="46"/>
      <g class="radar-phosphor">
        <path d="M31 33v34M43 25v50M56 30v40M68 38v24"/>
        <path class="radar-phosphor-link" d="M32 49h35"/>
      </g>
      <path class="radar-wire" d="M24 27v46m12-51v56m13-58v60m13-56v52m12-46v40M22 36h55M20 50h60M23 64h53"/>
      <path class="radar-glass-edge" d="M17 39a35 35 0 0 1 24-24m-18 27a29 29 0 0 1 7-13M67 84a38 38 0 0 0 17-17"/>
    </svg>
    <i class="radar-scanlines"></i>
  </span>`;

  window.lookFX = window.lookFX || {};
  window.lookFX.radar = {
    guide(u, ctx) {
      const node = ctx.ensureGuide(`radar-state-${u.id}`, "radar-state");
      const time = ctx.time || 0;
      if (!node._radar || time < node._radar.time) node._radar = { time, beam: null };
      node._radar.time = time;
      for (const fx of ctx.effects || []) {
        if (fx.name !== "beam" || fx.kind !== "雷达" || fx.unitId !== u.id) continue;
        node._radar.beam = { dx: fx.vx || 0, dy: fx.vy || 0, len: Math.max(0, fx.amount || 0) };
      }
      ctx.seenGuides.add(node.id);
    },
    unmount(el) {
      if (!el) return;
      el.querySelector(":scope > .radar-glass")?.remove();
      el.querySelector(":scope > .radar-beam")?.remove();
    },
    tick(el, u, ctx = {}) {
      if (!el || !el.classList.contains("look-radar")) return;
      let face = el.querySelector(":scope > .radar-glass");
      if (!face) {
        el.insertAdjacentHTML("beforeend", glass);
        face = el.querySelector(":scope > .radar-glass");
      }
      const time = ctx.time || 0;
      face.style.setProperty("--radar-phosphor", (0.84 + Math.sin(time * 2 * Math.PI / 3.8) * 0.025).toFixed(4));
      const st = document.getElementById(`radar-state-${u.id}`)?._radar?.beam;
      let beam = el.querySelector(":scope > .radar-beam");
      if (!st || st.len <= 0 || Math.hypot(st.dx, st.dy) < 1e-6) {
        beam?.remove();
        return;
      }
      if (!beam) {
        beam = document.createElement("i");
        beam.className = "radar-beam";
        beam.setAttribute("aria-hidden", "true");
        beam.innerHTML = '<i class="radar-tip"></i>';
        el.appendChild(beam);
      }
      // 世界 +Y 朝上，屏幕 Y 朝下。眩晕时沿用最后一次实际激光数据，不自算旋转。
      beam.style.setProperty("--ang", `${Math.atan2(-st.dy, st.dx)}rad`);
      beam.style.setProperty("--len", `${st.len * (ctx.scale || 1)}px`);
      const vision = u.vision > 0 ? u.vision : 9999;
      beam.querySelector(".radar-tip").hidden = st.len >= vision - 1e-4;
    },
  };
})();
