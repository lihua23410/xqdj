window.lookFX = window.lookFX || {};

const glitchEchoCount = 5;
const glitchEchoGap = 110;
const glitchTrailLife = glitchEchoCount * glitchEchoGap + 180;
const glitchTrails = (window.glitchTrails = window.glitchTrails || {});
const glitchEchoes = (window.glitchEchoes = window.glitchEchoes || {});

function glitchReach(el) {
  return (el.clientWidth || 36) * 0.5;
}

function glitchSnap(el) {
  const host = el.querySelector(":scope > .look-glitch-slices");
  if (!host) return [];
  const slices = [];
  for (const node of host.children) {
    if (node.style.display === "none") continue;
    slices.push({
      clip: node.style.clipPath,
      background: node.style.background,
      opacity: node.style.opacity,
      blend: node.style.mixBlendMode,
      transform: node.style.transform,
    });
  }
  return slices;
}

function glitchShift(transform) {
  const m = /translate\(\s*([-\d.]+)px/.exec(transform || "");
  return m ? parseFloat(m[1]) : 0;
}

function glitchRecord(u, now) {
  const id = String(u.id);
  let tr = glitchTrails[id];
  if (!tr) tr = glitchTrails[id] = [];
  const last = tr[tr.length - 1];
  if (!last || now - last.t >= 16) tr.push({ x: u.x, y: u.y, t: now });
  while (tr.length && now - tr[0].t > glitchTrailLife) tr.shift();
  return tr;
}

function glitchAt(samples, t) {
  const n = samples.length;
  if (!n || t < samples[0].t) return null;
  const last = samples[n - 1];
  if (t >= last.t) return last;
  for (let i = n - 1; i > 0; i--) {
    const a = samples[i];
    const b = samples[i - 1];
    if (t > a.t || t < b.t) continue;
    const span = a.t - b.t || 1;
    const u = (t - b.t) / span;
    return { x: b.x + (a.x - b.x) * u, y: b.y + (a.y - b.y) * u };
  }
  return null;
}

function glitchPaintSlices(box, slices, lag, reach, channel) {
  while (box.children.length < slices.length) box.appendChild(document.createElement("i"));
  while (box.children.length > slices.length) box.lastChild.remove();
  const amp = lag * reach * (channel ? 0.9 : 0.65);
  const fade = (0.78 - lag * 0.48).toFixed(3);
  slices.forEach((slice, n) => {
    const node = box.children[n];
    const wobble = (Math.random() - 0.5) * 2 * amp;
    node.style.clipPath = slice.clip;
    node.style.transform = `translate(${(glitchShift(slice.transform) + wobble).toFixed(1)}px, 0)`;
    node.style.background = channel === "c" ? "#3eefff" : channel === "r" ? "#ff3a4a" : slice.background;
    node.style.opacity = channel ? "1" : fade;
    node.style.mixBlendMode = "normal";
  });
}

function glitchPaintTrail(el, u, ctx) {
  const root = document.getElementById("fx");
  if (!root || typeof screenPos !== "function") return;
  const id = String(u.id);
  const now = ctx.now || performance.now();
  const samples = glitchRecord(u, now);
  let host = glitchEchoes[id];
  if (!host) {
    host = document.createElement("div");
    host.className = "glitch-echoes";
    root.appendChild(host);
    glitchEchoes[id] = host;
  }
  const kind = (el && el.style.getPropertyValue("--kind")) || "#4ec4ff";
  const slices = glitchSnap(el);
  const reach = glitchReach(el);
  const poses = [];
  for (let n = glitchEchoCount; n >= 1; n--) {
    const lag = n / glitchEchoCount;
    const p = glitchAt(samples, now - n * glitchEchoGap);
    if (!p || Math.hypot(p.x - u.x, p.y - u.y) < 10) continue;
    poses.push({ x: p.x, y: p.y, lag });
  }
  while (host.children.length < poses.length) {
    const echo = document.createElement("div");
    echo.className = "glitch-echo";
    const core = document.createElement("div");
    core.className = "core";
    const c = document.createElement("div");
    c.className = "chroma c";
    const m = document.createElement("div");
    m.className = "chroma m";
    echo.append(core, c, m);
    host.appendChild(echo);
  }
  while (host.children.length > poses.length) host.lastChild.remove();
  const d = Math.max(10, (u.radius || 18) * (ctx.scale || 1) * 2);
  poses.forEach((pose, i) => {
    const echo = host.children[i];
    const [sx, sy] = screenPos(pose.x, pose.y, ctx.scale, ctx.cx, ctx.cy);
    echo.style.left = `${sx}px`;
    echo.style.top = `${sy}px`;
    echo.style.width = `${d}px`;
    echo.style.height = `${d}px`;
    echo.style.opacity = (0.5 * (1 - pose.lag) + 0.12).toFixed(3);
    echo.style.setProperty("--kind", kind);
    echo.style.color = kind;
    echo.style.setProperty("--dx", `${(3 + pose.lag * 14).toFixed(1)}px`);
    glitchPaintSlices(echo.children[0], slices, pose.lag, reach, "");
    echo.children[1].style.display = "block";
    echo.children[2].style.display = "block";
    echo.children[1].style.opacity = (0.28 + pose.lag * 0.2).toFixed(3);
    echo.children[2].style.opacity = (0.22 + pose.lag * 0.42).toFixed(3);
    glitchPaintSlices(echo.children[1], slices, pose.lag, reach, "c");
    glitchPaintSlices(echo.children[2], slices, pose.lag, reach, "r");
  });
  if (!ctx.units) return;
  const live = new Set(ctx.units.map((o) => String(o.id)));
  live.add(id);
  for (const key of Object.keys(glitchEchoes)) {
    if (!live.has(key)) glitchDropTrail(key);
  }
}

function glitchDropTrail(id) {
  const host = glitchEchoes[id];
  if (host) host.remove();
  delete glitchEchoes[id];
  delete glitchTrails[id];
}

window.lookFX.glitch = {
  max: 28,
  modes: [
    "screen",
    "overlay",
    "difference",
    "exclusion",
    "lighter",
    "hard-light",
    "color-dodge",
    "multiply",
  ],
  unmount(el) {
    el?.querySelector(":scope > .look-glitch-slices")?.remove();
    if (el?.id) glitchDropTrail(el.id.replace(/^u-/, ""));
  },
  tick(el, u, ctx) {
    if (!el || !el.classList.contains("look-glitch")) return;
    let host = el.querySelector(":scope > .look-glitch-slices");
    if (!host) {
      host = document.createElement("div");
      host.className = "look-glitch-slices";
      for (let n = 0; n < this.max; n++) host.appendChild(document.createElement("i"));
      el.appendChild(host);
    }
    const kids = host.children;
    let y = 0;
    let i = 0;
    while (y < 100 && i < this.max) {
      const h = 1.5 + Math.random() * 12;
      const gap = Math.random() < 0.45 ? Math.random() * 10 : 0;
      const slice = kids[i++];
      const bottom = Math.max(0, 100 - y - h);
      slice.style.display = "block";
      slice.style.clipPath = `inset(${y.toFixed(2)}% 0 ${bottom.toFixed(2)}% 0)`;
      const reach = glitchReach(el);
      slice.style.transform = `translate(${((Math.random() - 0.5) * reach).toFixed(1)}px, 0)`;
      slice.style.mixBlendMode = this.modes[(Math.random() * this.modes.length) | 0];
      slice.style.opacity = String(0.65 + Math.random() * 0.35);
      slice.style.background =
        Math.random() < 0.22 ? (Math.random() < 0.5 ? "#0ff" : "#f0f") : "var(--kind)";
      y += h + gap;
    }
    for (; i < this.max; i++) kids[i].style.display = "none";
    if (u) glitchPaintTrail(el, u, ctx || {});
  },
};
