// 私有观众态跟随主人节点；所有脉动、眨眼与短效都按对局时间推进。
(function () {
  const hooks = window.lookFX = window.lookFX || {};
  const clamp = (n, lo, hi) => Math.max(lo, Math.min(hi, n));
  const number = n => Number.isFinite(Number(n)) ? Number(n) : 0;
  const count = n => Math.max(0, Math.round(number(n)));
  const stateNode = id => document.getElementById(`observer-state-${id}`);
  const state = id => stateNode(id)?._observer;
  const pulse = (now, at, duration) => clamp(1 - (now - at) / duration, 0, 1);

  function fresh(time) {
    return { time, key: "", gauge: 0, records: 0, tones: [], recordAt: [], targetId: 0,
      echo: false, beams: null, film: null, remaining: 0, total: 0,
      shotAt: -Infinity, lockAt: -Infinity, replayAt: -Infinity, effects: [] };
  }

  function ensure(el, cls, tag = "span") {
    let node = el.querySelector(`:scope > .${cls}`);
    if (!node) {
      node = document.createElement(tag);
      node.className = cls;
      el.appendChild(node);
    }
    return node;
  }

  function effect(root, data, fx, ctx, cls, life) {
    const node = document.createElement("i");
    node.className = `observer-fx ${cls}`;
    node.setAttribute("aria-hidden", "true");
    node.style.setProperty("--hue", String(number(fx.amount)));
    node.style.setProperty("--ang", `${Math.atan2(-number(fx.vy), number(fx.vx))}rad`);
    root.appendChild(node);
    data.effects.push({ node, at: ctx.time, life, x: fx.x, y: fx.y,
      capture: cls === "observer-capture" });
    // 密集弹幕下仅保留当前主人最近的短效，不影响胶片和队列读数。
    while (data.effects.length > 24) data.effects.shift().node.remove();
  }

  function updateEffects(data, ctx) {
    data.effects = data.effects.filter(e => {
      const progress = (ctx.time - e.at) / e.life;
      if (progress < 0 || progress >= 1) { e.node.remove(); return false; }
      const p = 1 - (1 - progress) ** 2;
      const x = e.capture ? e.x + (data.x - e.x) * p : e.x;
      const y = e.capture ? e.y + (data.y - e.y) * p : e.y;
      const [sx, sy] = arena.screenPos(x, y, ctx.scale, ctx.cx, ctx.cy);
      e.node.style.left = `${sx}px`;
      e.node.style.top = `${sy}px`;
      e.node.style.setProperty("--progress", progress.toFixed(4));
      e.node.style.setProperty("--fade", (1 - progress).toFixed(4));
      return true;
    });
  }

  function ingest(root, data, ctx, events) {
    for (const fx of events) {
      switch (fx.name) {
        case "observer-state": {
          data.gauge = clamp(number(fx.amount), 0, 100);
          const mark = count(fx.vx);
          data.records = mark % 4;
          data.echo = mark >= 4;
          data.targetId = count(fx.vy);
          // 前一轮颜色不能漏到新一轮；0 必须真的清空，而不是 slice(-0)。
          data.tones.length = data.records;
          break;
        }
        case "observer-film-state":
          data.beams = count(fx.amount);
          data.film = count(fx.vx);
          data.remaining = count(fx.vy);
          break;
        case "observer-lock":
          data.targetId = count(fx.vy);
          data.lockAt = ctx.time;
          effect(root, data, fx, ctx, "observer-lock-mark", .35);
          break;
        case "observer-record": {
          const index = clamp(count(fx.vx) - 1, 0, 2);
          data.tones[index] = number(fx.amount);
          data.recordAt[index] = ctx.time;
          data.records = Math.max(data.records, index + 1);
          effect(root, data, fx, ctx, "observer-capture", .42);
          break;
        }
        case "observer-replay":
          data.tones = [];
          data.records = 0;
          data.total = data.remaining = count(fx.amount);
          data.replayAt = ctx.time;
          break;
        case "observer-copy-shot":
        case "observer-beam-fire":
          data.shotAt = ctx.time;
          data.remaining = Math.max(0, data.remaining - 1);
          effect(root, data, fx, ctx, fx.name === "observer-beam-fire" ? "observer-spit beam" : "observer-spit", .18);
          break;
        case "observer-beam-hit":
        case "observer-kunai-hit":
          effect(root, data, fx, ctx, "observer-hit", .24);
          break;
      }
    }
  }

  hooks.observer = {
    guide(u, ctx) {
      const root = ctx.ensureGuide(`observer-state-${u.id}`, "observer-state");
      root.className = "observer-visual-state";
      const time = number(ctx.time);
      const events = (ctx.effects || []).filter(fx => fx.kind === u.kind && fx.unitId === u.id);
      const key = `${time}:${JSON.stringify(events)}`;
      if (!root._observer || time < root._observer.time) {
        root.replaceChildren();
        root._observer = fresh(time);
      }
      const data = root._observer;
      data.x = u.x; data.y = u.y;
      if (key !== data.key) {
        // 开局事件只在这一帧处理一次；重复导出同一帧不会重启短效。
        if (events.some(fx => fx.name === "observer-idle")) {
          root.replaceChildren();
          root._observer = fresh(time);
        }
        root._observer.x = u.x; root._observer.y = u.y;
        ingest(root, root._observer, { ...ctx, time }, events);
        root._observer.key = key;
        root._observer.x = u.x; root._observer.y = u.y;
      }
      const current = root._observer;
      current.time = time;
      updateEffects(current, { ...ctx, time });
      ctx.seenGuides.add(root.id);
      const target = (ctx.units || []).find(n => n.id === current.targetId);
      if (!target) return;
      const [x1, y1] = arena.screenPos(u.x, u.y, ctx.scale, ctx.cx, ctx.cy);
      const [x2, y2] = arena.screenPos(target.x, target.y, ctx.scale, ctx.cx, ctx.cy);
      const guide = ctx.ensureGuide(`observer-gaze-${u.id}`, "observer-gaze");
      guide.className = "guide observer-gaze";
      ctx.placeSeg(guide, x1, y1, x2, y2, u.kind);
      guide.style.opacity = String(.1 + current.gauge / 100 * .16);
      ctx.seenGuides.add(guide.id);
    },
    tick(el, u, ctx = {}) {
      if (!el?.classList.contains("look-observer")) return;
      const data = state(u.id) || fresh(number(ctx.time));
      const time = number(ctx.time);
      const target = (ctx.units || []).find(n => n.id === data.targetId);
      const replaying = !!target && data.remaining > 0;
      const shot = pulse(time, data.shotAt, .2);
      const breath = Math.sin(time * 2.1 + u.id * .7);
      const phase = (time + (u.id % 7) * .17) % 5.8;
      const blink = phase > 5.56 ? Math.sin((phase - 5.56) / .24 * Math.PI) : 0;
      el.classList.toggle("observer-locked", !!target);
      el.classList.toggle("observer-replaying", replaying);
      el.classList.toggle("observer-charged", data.gauge >= 85);
      el.style.setProperty("--observer-pulse", shot.toFixed(4));
      el.style.setProperty("--observer-breath", (1 + breath * .025).toFixed(4));
      el.style.setProperty("--observer-open", (1 - .94 * blink).toFixed(4));
      el.style.setProperty("--observer-ui-scale", String(clamp(number(ctx.scale) || 1, .8, 1.25)));

      const skin = ensure(el, "observer-skin");
      if (!skin.firstChild) skin.innerHTML = `<svg viewBox="0 0 100 100" aria-hidden="true"><path d="M7 38q13 2 19 12M10 31l6 12-9 9m87-13Q81 40 74 53m15-21-5 11 9 11M20 74q7-6 11-6m-6 7-3 11m48-16q6 3 10 12M37 81l-4 12m16-11 1 15m12-16 6 11"/></svg>`;
      const eye = ensure(el, "observer-eye");
      const vessels = ensure(eye, "observer-vessels");
      if (!vessels.firstChild) vessels.innerHTML = `<svg viewBox="0 0 100 60" aria-hidden="true"><path d="m2 18 12 7 11-1M13 25l-4 8m-7 11 16-6 9 1m71-20-14 6-10-2m11 2 5 9m8 10-16-7-9 3"/></svg>`;
      const iris = ensure(eye, "observer-iris");
      ensure(iris, "observer-pupil");
      const angle = target ? Math.atan2(-(target.y - u.y), target.x - u.x) : Math.sin(time * .6 + u.id) * .35;
      const reach = target ? Math.min(number(u.radius) * .2, 4.5) : 1;
      iris.style.transform = `translate(${(Math.cos(angle) * reach).toFixed(3)}px, ${(Math.sin(angle) * reach).toFixed(3)}px) scale(var(--observer-breath))`;

      const gauge = ensure(el, "observer-gauge-wrap");
      const label = ensure(gauge, "observer-gauge-label");
      label.textContent = "观测";
      ensure(gauge, "observer-gauge-value").textContent = `${Math.round(data.gauge)}%`;
      const track = ensure(gauge, "observer-gauge-track", "b");
      ensure(track, "observer-gauge-fill", "i").style.transform = `scaleX(${data.gauge / 100})`;
      const status = ensure(gauge, "observer-replay-status");
      status.textContent = replaying ? `复现 · ${data.remaining} 发` : "";
      status.hidden = !replaying;

      const films = ensure(el, "observer-films");
      for (let i = 0; i < 3; i++) {
        const card = ensure(films, `observer-film-${i}`);
        card.classList.add("observer-film");
        card.classList.toggle("active", i < data.records);
        card.style.setProperty("--hue", String(data.tones[i] ?? 185));
        card.style.setProperty("--fresh", pulse(time, data.recordAt[i] ?? -Infinity, .45).toFixed(4));
      }
      const echo = ensure(films, "observer-echo");
      echo.classList.toggle("active", data.echo || data.film > 0);
      echo.classList.toggle("has-beams", data.beams > 0);
      const counter = ensure(echo, "observer-echo-count");
      counter.textContent = data.film == null ? "" : `×${data.film}`;
      const description = data.film == null ? "特殊胶片" : `特殊胶片，共 ${data.film} 发，其中视线 ${data.beams} 发`;
      echo.setAttribute("aria-label", description);
      echo.setAttribute("title", description);
      echo.style.setProperty("--film-layers", String(Math.min(data.film || 0, 3)));
    },
    unmount(el) {
      for (const name of ["observer-skin", "observer-eye", "observer-gauge-wrap", "observer-films"]) {
        el?.querySelector(`:scope > .${name}`)?.remove();
      }
      el?.classList.remove("observer-locked", "observer-replaying", "observer-charged");
      for (const prop of ["--observer-pulse", "--observer-breath", "--observer-open", "--observer-ui-scale"]) el?.style.removeProperty(prop);
    },
  };

  hooks["observer-beam"] = {
    tick(el, u, ctx = {}) {
      const img = ensure(el, "observer-beam-img");
      img.style.transform = `rotate(${Math.atan2(-number(u.vy), number(u.vx))}rad)`;
      img.style.opacity = String(.86 + Math.sin(number(ctx.time) * 18) * .08);
    },
    unmount(el) { el?.querySelector(":scope > .observer-beam-img")?.remove(); },
  };
  hooks["observer-kunai-copy"] = {
    tick(el, u) { el.style.setProperty("--travel", `${Math.atan2(-number(u.vy), number(u.vx))}rad`); },
    unmount(el) { el?.style.removeProperty("--travel"); },
  };

  // 事件由 guide 按主人归档；注册空处理器，避免公共层生成同名占位短效。
  for (const name of ["idle", "state", "film-state", "lock", "record", "replay", "copy-shot", "beam-fire", "beam-hit", "kunai-hit"]) {
    arena.registerShot(`observer-${name}`, () => {});
  }

  const bodyColor = "#b9d7df";
  const ownShot = new Set(["观察者视线"]);

  function ensureBodyTint() {
    if (document.getElementById("observer-body-tint")) return;
    const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
    svg.setAttribute("aria-hidden", "true");
    svg.setAttribute("width", "0");
    svg.setAttribute("height", "0");
    svg.style.position = "absolute";
    svg.style.overflow = "hidden";
    const filter = document.createElementNS("http://www.w3.org/2000/svg", "filter");
    filter.id = "observer-body-tint";
    filter.setAttribute("color-interpolation-filters", "sRGB");
    const matrix = document.createElementNS("http://www.w3.org/2000/svg", "feColorMatrix");
    matrix.setAttribute("type", "matrix");
    // 先按 Rec.709 收成亮度，再乘上 #b9d7df。白趋于主体色，黑仍是黑。
    matrix.setAttribute("values", [
      "0.154249 0.518871 0.052380 0 0",
      "0.179251 0.603012 0.060875 0 0",
      "0.185921 0.625449 0.063140 0 0",
      "0 0 0 1 0",
    ].join(" "));
    filter.appendChild(matrix);
    svg.appendChild(filter);
    (document.body || document.documentElement).appendChild(svg);
    const style = document.createElement("style");
    style.textContent = '.ball.projectile[data-unit-paint="#b9d7df"]{filter:url("#observer-body-tint")}';
    document.head.appendChild(style);
  }

  ensureBodyTint();
  if (typeof arena.registerUnitPaint === "function") {
    arena.registerUnitPaint((u, units) => {
      if (!u || u.role !== "projectile" || ownShot.has(u.kind)) return "";
      const owner = (units || []).find((o) => o && o.id === u.ownerId);
      if (!owner || owner.kind !== "观察者") return "";
      return bodyColor;
    });
  }
})();
