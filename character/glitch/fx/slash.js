window.lookFX = window.lookFX || {};

const SLASH_WINDUP = 2;
const SLASH_FADE = 1;
const SLASH_LAYERS = [
  [110, [10, 58, 140, 42], true],     // 深蓝外晕
  [94, [10, 58, 140, 62], true],
  [78, [78, 196, 255, 80], true],     // 天蓝
  [62, [142, 232, 255, 130], true],
  [46, [223, 248, 255, 210], true],   // 近白芯
  [16, [10, 58, 140, 200], false, 0.82], // 深蓝内切
];

function rgba(rgb, a) {
  return `rgba(${rgb[0]},${rgb[1]},${rgb[2]},${Math.max(0, a) / 255})`;
}

function invScale(u) {
  const t = Math.max(0, Math.min(1, u));
  return (4 / (1 + 3 * t) - 1) / 3;
}

function drawLens(ctx, x1, y1, x2, y2, halfW, color, additive, lengthScale) {
  const dx = x2 - x1;
  const dy = y2 - y1;
  const len = Math.hypot(dx, dy);
  if (len < 1 || halfW < 0.5) return;
  const nx = -dy / len;
  const ny = dx / len;
  const mx = (x1 + x2) / 2;
  const my = (y1 + y2) / 2;
  const sx = mx - (dx * lengthScale) / 2;
  const sy = my - (dy * lengthScale) / 2;
  const ex = mx + (dx * lengthScale) / 2;
  const ey = my + (dy * lengthScale) / 2;
  ctx.globalCompositeOperation = additive ? "lighter" : "source-over";
  ctx.fillStyle = color;
  ctx.beginPath();
  ctx.moveTo(sx, sy);
  ctx.quadraticCurveTo(mx + nx * halfW, my + ny * halfW, ex, ey);
  ctx.quadraticCurveTo(mx - nx * halfW, my - ny * halfW, sx, sy);
  ctx.closePath();
  ctx.fill();
}

function spawnBurst(p1, p2, ox, oy) {
  const particles = [];
  const streaks = [];
  for (let i = 0; i < 35; i++) {
    const cyan = Math.random() >= 0.7;
    particles.push({
      x: ox,
      y: oy,
      vx: (Math.random() * 2 - 1) * 7,
      vy: (Math.random() * 2 - 1) * 7,
      life: 60 + Math.random() * 50,
      max: 0,
      size: 2 + Math.random() * 5,
      rgb: cyan ? [78, 196, 255] : [10, 58, 140],
    });
  }
  for (const p of particles) p.max = p.life;
  const dx = p2[0] - p1[0];
  const dy = p2[1] - p1[1];
  const lineLen = Math.hypot(dx, dy) || 1;
  const ux = dx / lineLen;
  const uy = dy / lineLen;
  const nx = -uy;
  const ny = ux;
  for (let i = 0; i < 20; i++) {
    const t = Math.random();
    const side = Math.random() < 0.5 ? 1 : -1;
    const init = 10 + Math.random() * 35;
    const ang = Math.atan2(ny * side, nx * side) + (Math.random() * 1.4 - 0.7);
    const spd = 1.5 + Math.random() * 2.5;
    const isDot = Math.random() < 0.8;
    streaks.push({
      x: p1[0] + dx * t + nx * side * init,
      y: p1[1] + dy * t + ny * side * init,
      vx: Math.cos(ang) * spd,
      vy: Math.sin(ang) * spd,
      angle: ang,
      isDot,
      radius: isDot ? 1.5 + Math.random() * 2 : 0,
      length: isDot ? 0 : 12 + Math.random() * 36,
      width: isDot ? 0 : 1.5 + Math.random() * 1.7,
      life: 70 + Math.random() * 30,
      max: 0,
    });
  }
  for (const s of streaks) s.max = s.life;
  return { particles, streaks };
}

function stepBurst(burst, frames) {
  for (const p of burst.particles) {
    p.x += p.vx * frames;
    p.y += p.vy * frames;
    p.life -= frames;
    p.vx *= Math.pow(0.97, frames);
    p.vy *= Math.pow(0.97, frames);
  }
  burst.particles = burst.particles.filter((p) => p.life > 0);
  for (const s of burst.streaks) {
    s.x += s.vx * frames;
    s.y += s.vy * frames;
    s.life -= frames;
    s.vx *= Math.pow(0.99, frames);
    s.vy *= Math.pow(0.99, frames);
  }
  burst.streaks = burst.streaks.filter((s) => s.life > 0);
}

function drawBurst(ctx, burst) {
  ctx.globalCompositeOperation = "source-over";
  for (const s of burst.streaks) {
    const ratio = s.life / s.max;
    const a = Math.max(0, ratio);
    if (a <= 0) continue;
    ctx.save();
    ctx.translate(s.x, s.y);
    ctx.rotate(s.angle);
    if (s.isDot) {
      ctx.fillStyle = `rgba(78,196,255,${0.27 * a})`;
      ctx.beginPath();
      ctx.ellipse(0, 0, s.radius * 2, s.radius * 2, 0, 0, Math.PI * 2);
      ctx.fill();
      ctx.fillStyle = `rgba(223,248,255,${a})`;
      ctx.beginPath();
      ctx.ellipse(0, 0, s.radius, s.radius, 0, 0, Math.PI * 2);
      ctx.fill();
    } else {
      const drawShuttle = (w, length, alpha, rgb) => {
        const h = w / 2;
        ctx.fillStyle = `rgba(${rgb},${alpha})`;
        ctx.beginPath();
        ctx.moveTo(0, 0);
        ctx.quadraticCurveTo(length / 2, -h, length, 0);
        ctx.quadraticCurveTo(length / 2, h, 0, 0);
        ctx.closePath();
        ctx.fill();
      };
      drawShuttle(s.width * 1.6, s.length + 6, 0.27 * a, "78,196,255");
      drawShuttle(s.width, s.length, a, "223,248,255");
    }
    ctx.restore();
  }
  for (const p of burst.particles) {
    const ratio = p.life / p.max;
    const a = ratio ** 0.7;
    const size = Math.max(1, p.size * ratio);
    ctx.fillStyle = rgba(p.rgb, 255 * a);
    ctx.fillRect(p.x, p.y, size, size);
  }
}

const slashState = new WeakMap();

window.lookFX.slash = {
  unmount(el) {
    slashState.delete(el);
    el?.querySelector(":scope > .look-iai")?.remove();
    el?.querySelector(":scope > .slash-ryo")?.remove();
    document.querySelector(".slash-flash")?.remove();
    const stage = document.querySelector(".stage");
    if (stage) stage.style.transform = "";
  },
  tick(el, u) {
    if (!el || !el.classList.contains("look-slash")) return;
    const hexEl = document.getElementById("hex");
    const stageW = hexEl ? hexEl.clientWidth : 560;
    const stageH = hexEl ? hexEl.clientHeight : 485;
    const scale = stageW / 560;
    const diam = 36 * scale;
    const grow = Math.hypot(stageW, stageH) / stageW; // 斜向铺满时的等比例放大
    const len = Math.hypot(stageW, stageH) * 2.2;
    const maxW = diam * 5 * grow;
    const ang = Math.atan2(-(u.vy || 0), u.vx || 1);

    let host = el.querySelector(":scope > .look-iai");
    if (!host) {
      host = document.createElement("div");
      host.className = "look-iai";
      for (const name of ["iai-glow", "iai-core", "iai-flare"]) {
        const layer = document.createElement("i");
        layer.className = name;
        host.appendChild(layer);
      }
      el.appendChild(host);
    }
    host.style.setProperty("--slash-ang", `${ang}rad`);
    host.style.setProperty("--slash-short", `${diam * 2.5}px`);

    let canvas = el.querySelector(":scope > .slash-ryo");
    if (!canvas) {
      canvas = document.createElement("canvas");
      canvas.className = "slash-ryo";
      el.appendChild(canvas);
    }

    let st = slashState.get(el);
    const now = performance.now();
    if (!st) {
      st = { t0: now, last: now, burst: null, flashed: false };
      slashState.set(el, st);
    }
    const t = (now - st.t0) / 1000;
    const dt = Math.min(0.05, (now - st.last) / 1000);
    st.last = now;

    if (t < SLASH_WINDUP) {
      host.style.display = "";
      canvas.style.display = "none";
      return;
    }

    host.style.display = "none";
    canvas.style.display = "block";
    const fadeU = (t - SLASH_WINDUP) / SLASH_FADE;
    const bladeScale = invScale(fadeU);
    const dpr = window.devicePixelRatio || 1;
    const cw = Math.ceil(len + maxW + 80);
    const ch = cw;
    canvas.style.width = `${cw}px`;
    canvas.style.height = `${ch}px`;
    if (canvas.width !== Math.ceil(cw * dpr) || canvas.height !== Math.ceil(ch * dpr)) {
      canvas.width = Math.ceil(cw * dpr);
      canvas.height = Math.ceil(ch * dpr);
    }
    const ctx = canvas.getContext("2d");
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.clearRect(0, 0, cw, ch);

    const ox = cw / 2;
    const oy = ch / 2;
    const ux = Math.cos(ang);
    const uy = Math.sin(ang);
    const p1 = [ox - ux * (len / 2), oy - uy * (len / 2)];
    const p2 = [ox + ux * (len / 2), oy + uy * (len / 2)];

    if (!st.burst) {
      st.burst = spawnBurst(p1, p2, ox, oy);
      if (!st.flashed) {
        st.flashed = true;
        document.querySelector(".slash-flash")?.remove();
        const stage = document.querySelector(".stage");
        if (stage) {
          const flash = document.createElement("div");
          flash.className = "slash-flash";
          stage.appendChild(flash);
          flash.addEventListener("animationend", () => flash.remove());
        }
      }
    }

    const widthK = maxW / 110;
    if (bladeScale > 0.01) {
      for (const layer of SLASH_LAYERS) {
        const [fullW, rgb, additive] = layer;
        const lengthScale = layer[3] ?? 1;
        const half = (fullW * widthK * bladeScale) / 2;
        const alpha = rgb[3] * bladeScale;
        drawLens(ctx, p1[0], p1[1], p2[0], p2[1], half, rgba(rgb, alpha), additive, lengthScale);
      }
    }

    const frames = dt * 60;
    stepBurst(st.burst, frames);
    drawBurst(ctx, st.burst);

    const stage = document.querySelector(".stage");
    if (stage) {
      const burstT = t - SLASH_WINDUP;
      if (burstT < 0.18) {
        const mag = 7 * (1 - burstT / 0.18);
        stage.style.transform = `translate(${(Math.random() * 2 - 1) * mag}px, ${(Math.random() * 2 - 1) * mag}px)`;
      } else if (stage.style.transform) {
        stage.style.transform = "";
      }
    }
  },
};
