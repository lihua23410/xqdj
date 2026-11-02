// Coverage belongs to this pack. The floor and the advancing flame share it.
window.lookFX = window.lookFX || {};

(function () {
  if (window.arena && window.arena.registerShot) {
    // These are snapshots of persistent geometry, not one-shot DOM animations.
    for (const name of ["arson-surface", "arson-burning", "arson-oil", "arson-fire"]) {
      window.arena.registerShot(name, () => {});
    }
  }

  function hash(x, y) {
    let n = Math.imul(x, 374761393) ^ Math.imul(y, 668265263);
    n = Math.imul(n ^ (n >>> 13), 1274126177);
    return ((n ^ (n >>> 16)) >>> 0) / 4294967295;
  }

  function canvasIn(el, cls) {
    const canvas = document.createElement("canvas");
    canvas.className = cls;
    canvas.setAttribute("aria-hidden", "true");
    el.appendChild(canvas);
    return canvas;
  }

  function paint(canvas, surface) {
    const side = surface.side;
    let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
    for (const chunk of surface.chunks) {
      for (const runs of [chunk.o || [], chunk.f || []]) {
        for (let i = 0; i < runs.length; i += 2) {
          const x = chunk.x * side + runs[i] % side;
          const y = chunk.y * side + Math.floor(runs[i] / side);
          minX = Math.min(minX, x); maxX = Math.max(maxX, x + runs[i + 1]);
          minY = Math.min(minY, y); maxY = Math.max(maxY, y + 1);
        }
      }
    }
    if (!Number.isFinite(minX)) {
      canvas.width = canvas.height = 0;
      return null;
    }
    minX--; minY--; maxX++; maxY++;
    const width = maxX - minX, height = maxY - minY;
    const cells = new Uint8Array(width * height);
    for (const chunk of surface.chunks) {
      for (const [state, runs] of [[1, chunk.o || []], [2, chunk.f || []]]) {
        for (let i = 0; i < runs.length; i += 2) {
          const x = chunk.x * side + runs[i] % side - minX;
          const y = maxY - 1 - chunk.y * side - Math.floor(runs[i] / side);
          cells.fill(state, y * width + x, y * width + x + runs[i + 1]);
        }
      }
    }
    canvas.width = width; canvas.height = height;
    const context = canvas.getContext("2d");
    const image = context.createImageData(width, height);
    const fire = [];
    for (let y = 1; y < height - 1; y++) {
      for (let x = 1; x < width - 1; x++) {
        const i = y * width + x, state = cells[i];
        if (!state) continue;
        const up = cells[i - width], down = cells[i + width];
        const near = [cells[i - 1], cells[i + 1], up, down];
        const edge = near.includes(0);
        const front = state === 2 && (near.includes(1) ||
          cells[i - width - 1] === 1 || cells[i - width + 1] === 1 ||
          cells[i + width - 1] === 1 || cells[i + width + 1] === 1);
        const wx = x + minX, wy = maxY - y;
        const grain = hash(wx, wy);
        let color;
        if (state === 1) {
          // Pitch-black tar, a raised meniscus, and narrow, bent reflections.
          const wave = Math.max(0, Math.sin(wx * .13 + Math.sin(wy * .11) * 1.8));
          const sheen = Math.pow(wave, 28) * (.3 + .7 * hash(wx >> 3, wy >> 3));
          if (!up) color = [44, 47, 49, 248];
          else if (!down) color = [6, 7, 9, 252];
          else color = [11 + sheen * 30 + grain * 2, 12 + sheen * 33, 15 + sheen * 37, 250];
        } else {
          // This is the coal bed; moving tongues are a separate bounded canvas.
          color = front ? [255, 218, 117, 255] : edge ? [130, 33, 9, 220] : [174 + grain * 20, 48 + grain * 17, 10, 238];
          fire.push({ x: x + .5, y: y + .5, front });
        }
        image.data.set(color, i * 4);
      }
    }
    context.putImageData(image, 0, 0);
    const stride = Math.max(1, Math.ceil(fire.length / 72));
    return { minX, maxY, width, height, fire: fire.filter((p, i) => i % stride === 0).slice(0, 72) };
  }

  function tongue(c, x, y, width, height, phase) {
    const bend = Math.sin(phase * 1.7) * width * .45;
    const gradient = c.createLinearGradient(x, y, x + bend, y - height);
    gradient.addColorStop(0, "rgba(172,31,5,.1)");
    gradient.addColorStop(.23, "rgba(246,72,9,.8)");
    gradient.addColorStop(.65, "rgba(255,164,30,.95)");
    gradient.addColorStop(1, "rgba(255,231,139,.5)");
    c.fillStyle = gradient;
    c.beginPath();
    c.moveTo(x - width * .5, y);
    c.bezierCurveTo(x - width, y - height * .4, x + bend - width * .2, y - height * .58, x + bend, y - height);
    c.bezierCurveTo(x + bend + width * .12, y - height * .62, x + width, y - height * .35, x + width * .5, y);
    c.closePath(); c.fill();
    c.fillStyle = "rgba(255,207,81,.85)";
    c.beginPath();
    c.moveTo(x - width * .24, y - height * .08);
    c.bezierCurveTo(x - width * .35, y - height * .3, x + bend * .3, y - height * .36, x + bend * .4, y - height * .64);
    c.bezierCurveTo(x + width * .12, y - height * .35, x + width * .4, y - height * .17, x + width * .24, y - height * .08);
    c.closePath(); c.fill();
  }

  function paintFire(st, time) {
    const b = st.bounds;
    if (!b || !b.fire.length) {
      if (st.flames) st.flames.width = st.flames.height = 0;
      return;
    }
    if (!st.flames) st.flames = canvasIn(st.el, "arson-flames");
    const frame = Math.floor(time * 20);
    if (st.flameFrame === frame) return;
    st.flameFrame = frame;
    const canvas = st.flames, c = canvas.getContext("2d"), pad = 12;
    const width = b.width * 2, height = (b.height + pad) * 2;
    if (canvas.width !== width || canvas.height !== height) {
      canvas.width = width; canvas.height = height;
      canvas.style.height = `${(b.height + pad) / b.height * 100}%`;
      canvas.style.top = `${-pad / b.height * 100}%`;
    }
    c.clearRect(0, 0, width, height);
    const t = frame / 20;
    for (let i = 0; i < b.fire.length; i++) {
      const p = b.fire[i], seed = hash(p.x + b.minX, b.maxY - p.y);
      const phase = t * 6 + seed * 30;
      const pulse = .82 + Math.sin(phase) * .18;
      const w = (2.7 + seed * 2.2) * 2;
      const h = (4.5 + seed * 4.5) * 2 * pulse;
      tongue(c, p.x * 2, (p.y + pad) * 2, w, h, phase);
      if (i % 9 === 0) {
        const rise = (t * .9 + seed) % 1;
        c.fillStyle = `rgba(255,190,70,${(1 - rise) * .7})`;
        c.fillRect(p.x * 2 + Math.sin(phase) * 3, (p.y + pad) * 2 - rise * 28, 1, 2);
      }
    }
  }

  function paintBurns(burns, ctx) {
    for (const burn of burns || []) {
      if (burn.until + 1e-9 < ctx.time) continue;
      const target = (ctx.units || []).find(o => o.id === burn.id && o.hp > 0);
      if (!target || !(target.radius > 0)) continue;
      // Shared key: multiple arsonists never stack a pile of visual nodes.
      const key = `arson-burning-${target.id}`;
      const el = ctx.ensureGuide(key, "arson-burning");
      ctx.seenGuides.add(key);
      let st = el.arsonBurn;
      if (!st) {
        const canvas = canvasIn(el, "arson-burn-flames");
        canvas.width = 96; canvas.height = 120;
        st = el.arsonBurn = { canvas, frame: -1 };
      }
      const r = Math.max(5, target.radius) * ctx.scale;
      const layout = `${target.x}/${target.y}/${r}/${ctx.cx}/${ctx.cy}`;
      if (st.layout !== layout) {
        el.style.width = `${r * 3.2}px`; el.style.height = `${r * 4}px`;
        el.style.left = `${ctx.cx + target.x * ctx.scale - r * 1.6}px`;
        el.style.top = `${ctx.cy - target.y * ctx.scale - r * 2.55}px`;
        st.layout = layout;
      }
      const frame = Math.floor(ctx.time * 20);
      if (st.frame === frame) continue;
      st.frame = frame;
      const c = st.canvas.getContext("2d"), t = frame / 20;
      c.clearRect(0, 0, 96, 120);
      const glow = c.createRadialGradient(48, 79, 12, 48, 79, 37);
      glow.addColorStop(0, "rgba(255,80,10,0)");
      glow.addColorStop(.65, "rgba(255,92,14,.28)");
      glow.addColorStop(1, "rgba(255,80,10,0)");
      c.fillStyle = glow; c.fillRect(8, 40, 80, 76);
      for (let i = 0; i < 7; i++) {
        const a = i / 7 * Math.PI * 2, seed = hash(target.id, i);
        const phase = t * 7 + seed * 17;
        tongue(c, 48 + Math.cos(a) * 23, 79 + Math.sin(a) * 13,
          12 + seed * 5, (24 + seed * 17) * (.85 + Math.sin(phase) * .15), phase);
      }
      for (let i = 0; i < 4; i++) {
        const seed = hash(target.id, i + 10), rise = (t * .8 + seed) % 1;
        c.fillStyle = `rgba(255,205,91,${1 - rise})`;
        c.fillRect(26 + seed * 44 + Math.sin(t * 3 + i) * 4, 65 - rise * 45, 1.5, 3);
      }
    }
  }

  window.lookFX["arson-field"] = {
    guide(u, ctx) {
      if (!ctx || !(ctx.scale > 0) || !ctx.seenGuides || !ctx.ensureGuide) return;
      const key = `arson-surface-${u.id}`;
      const el = ctx.ensureGuide(key, "arson-surface");
      ctx.seenGuides.add(key);
      let st = el.arsonSurface;
      if (!st) st = el.arsonSurface = { el, canvas: canvasIn(el, "arson-floor"), revision: -1, time: -Infinity, burns: [] };
      let surface, burns;
      for (const fx of ctx.effects || []) {
        if (!fx || fx.unitId !== u.id) continue;
        if (fx.name === "arson-surface" && fx.data) surface = fx.data;
        if (fx.name === "arson-burning" && fx.data) burns = fx.data;
      }
      if (ctx.time < st.time) { st.burns = []; st.flameFrame = null; }
      if (burns) st.burns = burns;
      if (surface && (st.revision !== surface.revision || ctx.time < st.time)) {
        st.bounds = paint(st.canvas, surface);
        st.revision = surface.revision; st.step = surface.step;
        st.layout = null; st.flameFrame = null;
      } else if (!surface && ctx.time < st.time) {
        st.canvas.width = st.canvas.height = 0;
        st.bounds = null; st.revision = -1;
      }
      st.time = ctx.time;
      paintFire(st, ctx.time);
      paintBurns(st.burns, ctx);
      if (!st.bounds) return;
      const layout = `${ctx.scale}/${ctx.cx}/${ctx.cy}`;
      if (st.layout === layout) return;
      const b = st.bounds, scale = st.step * ctx.scale;
      el.style.width = `${b.width * scale}px`; el.style.height = `${b.height * scale}px`;
      el.style.left = `${ctx.cx + b.minX * scale}px`; el.style.top = `${ctx.cy - b.maxY * scale}px`;
      st.layout = layout;
    },
  };

  window.lookFX.arson = {
    tick(el, u, ctx) {
      const frame = Math.floor(ctx.time * 20);
      if (el.arsonEmberFrame === frame) return;
      el.arsonEmberFrame = frame;
      el.style.setProperty("--ember-pulse", .8 + Math.sin(frame / 9 + u.id) * .2);
    },
  };
  function bottleTick(el, u, ctx, lit) {
    let bottle = el.querySelector(":scope > .arson-bottle");
    if (!bottle) {
      bottle = document.createElement("span"); bottle.className = "arson-bottle";
      if (lit) {
        const wick = document.createElement("i"); wick.className = "arson-wick"; bottle.appendChild(wick);
      }
      el.appendChild(bottle);
    }
    const frame = Math.floor(ctx.time * 20);
    if (el.arsonBottleFrame === frame) return;
    el.arsonBottleFrame = frame;
    const time = frame / 20;
    bottle.style.setProperty("--bottle-tilt", `${Math.sin(time * 4 + u.id) * .3 + Math.atan2(-(u.vy || 0), u.vx || 1) * .2}rad`);
    bottle.style.setProperty("--wick-height", .85 + Math.sin(time * 14 + u.id) * .15);
  }
  window.lookFX["arson-oil"] = { tick(el, u, ctx) { bottleTick(el, u, ctx, false); } };
  window.lookFX["arson-fire"] = { tick(el, u, ctx) { bottleTick(el, u, ctx, true); } };
})();
