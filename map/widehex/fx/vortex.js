(function () {
  window.lookFX = window.lookFX || {};

  // 点的位置来自 tsparticles（MIT，约 9k star）的 drawSpiral，不引入这个库。
  // https://github.com/tsparticles/tsparticles/blob/d38e87725cb0fa7108481b39064e067203068bac/shapes/spiral/src/Utils.ts
  // 源码：angle = i / 10，r = innerRadius + lineSpacing * angle，再 lineTo(r cos, r sin)。
  // 源码只描一根细线，这里用同一公式画多条旋臂，并让亮点沿臂往涡眼走。
  const widthFactor = 48;
  const arms = 2;

  // 源码 widthFactor 取 10，一步的弦长和线宽差不多，56px 上会断成珠子。
  // 曲线仍是 r = inner + spacing * angle，只把角步加密。
  function spiralPoint(inner, spacing, angle, spin, arm) {
    const factor = inner + spacing * angle;
    const a = angle + spin + (arm * Math.PI * 2) / arms;
    return {
      x: factor * Math.cos(a),
      y: factor * Math.sin(a),
      factor: factor,
    };
  }

  function trace(g, inner, spacing, maxR, spin, arm) {
    const realWidth = (maxR - inner) / spacing;
    const n = realWidth * widthFactor;
    g.beginPath();
    for (let i = 0; i <= n; i++) {
      const p = spiralPoint(inner, spacing, i / widthFactor, spin, arm);
      if (i === 0) g.moveTo(p.x, p.y);
      else g.lineTo(p.x, p.y);
    }
  }

  function strokeBand(g, inner, spacing, maxR, spin, width, color, blur) {
    g.lineJoin = "round";
    g.lineCap = "round";
    g.strokeStyle = color;
    g.lineWidth = width;
    g.shadowColor = color;
    g.shadowBlur = blur;
    for (let arm = 0; arm < arms; arm++) {
      trace(g, inner, spacing, maxR, spin, arm);
      g.stroke();
    }
    g.shadowBlur = 0;
  }

  function specks(g, inner, spacing, maxR, spin, travel, color) {
    const n = ((maxR - inner) / spacing) * 10;
    g.fillStyle = color;
    for (let arm = 0; arm < arms; arm++) {
      for (let k = 0; k < 28; k++) {
        const i = ((1 - ((k / 28 + travel) % 1)) % 1) * n;
        const p = spiralPoint(inner, spacing, i / 10, spin, arm);
        g.beginPath();
        g.arc(p.x, p.y, Math.max(0.45, maxR * 0.012), 0, Math.PI * 2);
        g.fill();
      }
    }
  }

  window.lookFX.vortex = {
    tick(el, _u, ctx) {
      if (!el) return;
      let canvas = el.querySelector(":scope > .vortex-spiral");
      if (!canvas) {
        canvas = document.createElement("canvas");
        canvas.className = "vortex-spiral";
        el.appendChild(canvas);
      }
      const size = el.clientWidth;
      if (!(size > 0)) return;
      const dpr = window.devicePixelRatio || 1;
      const px = Math.max(1, Math.round(size * dpr));
      if (canvas.width !== px) {
        canvas.width = px;
        canvas.height = px;
      }
      const g = canvas.getContext("2d");
      const radius = px / 2;
      const R = radius * 0.96;
      const now = (ctx && ctx.now) || 0;
      const outerSpin = (-now / 2400) * Math.PI * 2;
      const innerSpin = (-now / 900) * Math.PI * 2;

      g.setTransform(1, 0, 0, 1, 0, 0);
      g.clearRect(0, 0, px, px);
      g.translate(radius, radius);

      const wash = g.createRadialGradient(0, 0, R * 0.18, 0, 0, R);
      wash.addColorStop(0, "rgba(4, 16, 24, 0)");
      wash.addColorStop(0.5, "rgba(18, 78, 98, 0.45)");
      wash.addColorStop(0.82, "rgba(42, 144, 184, 0.28)");
      wash.addColorStop(1, "rgba(78, 200, 224, 0)");
      g.fillStyle = wash;
      g.beginPath();
      g.arc(0, 0, R, 0, Math.PI * 2);
      g.fill();

      strokeBand(g, R * 0.22, R * 0.055, R, outerSpin, R * 0.09, "rgba(42, 144, 184, 0.55)", R * 0.04);
      strokeBand(g, R * 0.22, R * 0.055, R, outerSpin, R * 0.025, "rgba(122, 232, 255, 0.95)", 0);
      strokeBand(g, R * 0.16, R * 0.04, R * 0.5, innerSpin, R * 0.055, "rgba(78, 200, 224, 0.8)", R * 0.03);
      strokeBand(g, R * 0.16, R * 0.04, R * 0.5, innerSpin, R * 0.016, "rgba(233, 251, 255, 0.95)", 0);

      specks(g, R * 0.22, R * 0.055, R, outerSpin, (now % 1700) / 1700, "rgba(233, 251, 255, 0.85)");
      specks(g, R * 0.16, R * 0.04, R * 0.5, innerSpin, (now % 700) / 700, "rgba(255, 255, 255, 0.9)");
    },
    unmount(el) {
      el.querySelector(":scope > .vortex-spiral")?.remove();
    },
  };
})();
