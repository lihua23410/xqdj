(function () {
  window.lookFX = window.lookFX || {};
  // 轨迹留在随弹清理的 guide 上，只记录真实快照；对局时间不变就不采样。
  function flight(u, ctx) {
    const node = ctx.ensureGuide(`flicker-flight-${u.id}`, "flicker-flight");
    ctx.seenGuides.add(node.id);
    const time = ctx.time || 0;
    let track = node._flight;
    if (!track || time < track.time || time - track.time > .2) {
      track = node._flight = { time, points: [] };
    }
    const last = track.points.at(-1);
    if (last && Math.hypot(u.x - last.x, u.y - last.y) > 40 + Math.hypot(u.vx || 0, u.vy || 0) * Math.max(0, time - last.t)) {
      track.points.length = 0;
    }
    if (!track.points.length || time > track.time + 1e-6) track.points.push({ x: u.x, y: u.y, t: time });
    track.time = time;
    track.points = track.points.filter(p => time - p.t <= .14).slice(-12);
    if (!node._canvas) { node._canvas = document.createElement("canvas"); node.appendChild(node._canvas); }
    const scale = ctx.scale || 1, size = 192 * scale;
    const ratio = Math.min(2, window.devicePixelRatio || 1);
    const canvas = node._canvas, pixels = Math.ceil(size * ratio);
    if (canvas.width !== pixels) { canvas.width = pixels; canvas.height = pixels; }
    canvas.style.width = `${size}px`; canvas.style.height = `${size}px`;
    node.style.left = `${ctx.cx + u.x * scale}px`;
    node.style.top = `${ctx.cy - u.y * scale}px`;
    node.style.width = `${size}px`; node.style.height = `${size}px`;
    const g = canvas.getContext("2d");
    if (!g) return;
    g.setTransform(ratio, 0, 0, ratio, 0, 0);
    g.clearRect(0, 0, size, size);
    g.lineCap = "round";
    const boosted = u.kind === "闪烁强化智能弹";
    const split = u.kind === "闪烁折线弹";
    const color = boosted ? "#ffbb55" : split ? "#c9aaff" : "#59eacb";
    const pts = track.points;
    // 分段变细变淡，弯曲来自真实转弯。头部短亮、尾部透明，不堆圆形残影。
    for (let i = 1; i < pts.length; i++) {
      const a = pts[i - 1], b = pts[i];
      const fade = Math.max(0, 1 - (time - a.t) / .14);
      const ax = size / 2 + (a.x - u.x) * scale, ay = size / 2 - (a.y - u.y) * scale;
      const bx = size / 2 + (b.x - u.x) * scale, by = size / 2 - (b.y - u.y) * scale;
      for (const [width, alpha, tint, offset] of [
        [3.5 * fade, .12 * fade, color, 0],
        [1.25 * fade, .65 * fade, color, 0],
        [.6 * fade, .35 * fade, boosted ? "#fff2b5" : "#b5a0ff", 1.3],
      ]) {
        g.lineWidth = Math.max(.2, width) * scale;
        g.globalAlpha = alpha; g.strokeStyle = tint;
        g.beginPath(); g.moveTo(ax, ay + offset * scale); g.lineTo(bx, by + offset * scale); g.stroke();
      }
    }
    g.globalAlpha = 1;
  }
  function tick(el, u, ctx) {
    if (!el) return;
    if (!el.querySelector(":scope > .flicker-round")) {
      const round = document.createElement("span"); round.className = "flicker-round";
      for (const name of ["streak", "core"]) {
        const part = document.createElement("i"); part.className = `flicker-round-${name}`; round.appendChild(part);
      }
      el.appendChild(round);
    }
    el.style.setProperty("--flicker-heading", `${Math.atan2(-(u.vy || 0), u.vx || 0)}rad`);
    el.style.setProperty("--flicker-scale", ctx.scale || 1);
  }
  function unmount(el) { el?.querySelector(":scope > .flicker-round")?.remove(); }
  window.lookFX["flicker-bullet"] = { tick, unmount };
  window.lookFX["flicker-split"] = { tick, unmount, guide: flight };
  window.lookFX["flicker-smart"] = { tick, unmount, guide: flight };
  window.lookFX["flicker-boost"] = { tick, unmount, guide: flight };
})();
