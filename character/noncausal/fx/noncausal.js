/* 非因果观众呈现。字段跟 fx/shot.js 顶上的对照走，不另起一套协议。
   直方图只在 spectrum / spectrum-bin 上立起来；开火同帧点亮 binIndex。
   倒带最多 3 秒，只画已经看见的轨，不倒放粒子。 */
(function () {
  var root = typeof window !== "undefined" ? window : globalThis;
  root.lookFX = root.lookFX || {};

  var CAPTION = "打的是对面那条死掉的世界线";
  var GHOST_OPACITY = 0.3;
  var REWIND_MAX_SEC = 3;
  var FLASH_MERGE_MS = 90;
  var THICK_MIN = 0.72;
  var THICK_MAX = 1.85;
  var RESIDUAL = "残影";
  var LOCK = "锁影弹";

  function num(v) {
    var n = Number(v);
    return Number.isFinite(n) ? n : null;
  }

  function createState() {
    return {
      bins: [],
      binCount: 0,
      pulses: 0,
      cadence: 0,
      doomAt: 0,
      doomStart: null,
      ghostMax: 0,
      ghostHP: null,
      hot: -1,
      drawnAmount: 0,
      pending: null,
      lastFlashAt: -1e15,
      skipped: false,
      introT0: null,
      tracking: true,
      frozen: false,
      track: [],
      frozenTrack: [],
      doomed: false,
      down: false,
      breaks: 0,
      hits: 0,
    };
  }

  var buckets = {};
  var owners = {};
  var missiles = {};
  var tornAt = {};

  function stateFor(id) {
    var k = String(id == null ? 0 : id);
    if (!buckets[k]) buckets[k] = createState();
    return buckets[k];
  }

  function reset() {
    buckets = {};
    owners = {};
    missiles = {};
    tornAt = {};
  }

  function bindResidual(id, ownerId) {
    if (id == null || ownerId == null) return;
    owners[String(id)] = ownerId;
  }

  function markTorn(id, now) {
    if (id == null) return;
    tornAt[String(id)] = now + 350;
  }

  function torn(id, nowMs) {
    return (tornAt[String(id)] || 0) > (nowMs || 0);
  }

  function simNow() {
    if (typeof state !== "undefined" && state && Number.isFinite(Number(state.time))) {
      return Number(state.time);
    }
    var s = root.state;
    if (s && Number.isFinite(Number(s.time))) return Number(s.time);
    return 0;
  }

  function thicknessForTier(tier, n) {
    var count = n > 0 ? n : 1;
    var t = Math.max(0, Math.min(count - 1, Math.round(Number(tier) || 0)));
    if (count <= 1) return 1;
    return THICK_MIN + (THICK_MAX - THICK_MIN) * (t / (count - 1));
  }

  function maxAmount(st) {
    var max = null;
    var bins = st.bins || [];
    for (var i = 0; i < bins.length; i++) {
      if (!bins[i]) continue;
      if (max == null || bins[i].amount > max) max = bins[i].amount;
    }
    return max;
  }

  function isBig(st, bin, amount) {
    var max = maxAmount(st);
    if (max != null) return amount + 1e-9 >= max && max > 0 && (st.binCount || st.bins.length) > 1;
    return (st.binCount || 0) > 1 && bin >= st.binCount - 1;
  }

  function noteFire(st, bin, amount, nowMs) {
    st.hot = bin;
    st.drawnAmount = amount;
    var n = Math.max(st.bins.length, st.binCount, bin + 1, 1);
    var thick = thicknessForTier(bin, n);
    st.pending = { bin: bin, amount: amount, thick: thick };
    var flash = "none";
    if (isBig(st, bin, amount)) {
      flash = nowMs - st.lastFlashAt < FLASH_MERGE_MS ? "merge" : "spawn";
      st.lastFlashAt = nowMs;
    }
    return { kind: "fire", flash: flash, thick: thick, binIndex: bin, drawnAmount: amount };
  }

  function applyEffect(st, fx, nowMs, sim) {
    var name = fx && fx.name;
    var now = nowMs == null ? 0 : nowMs;
    var t = sim == null ? 0 : sim;
    if (!st || !fx) return { kind: "", flash: "none" };
    if (name === "spectrum") {
      st.cadence = num(fx.amount) || 0;
      st.binCount = Math.max(0, Math.round(num(fx.vx) || 0));
      st.pulses = Math.round(num(fx.vy) || 0);
      while (st.bins.length < st.binCount) st.bins.push(null);
      return { kind: "spectrum", flash: "none" };
    }
    if (name === "spectrum-bin") {
      var idx = Math.round(num(fx.slot) || 0);
      if (idx < 0) idx = 0;
      while (st.bins.length <= idx) st.bins.push(null);
      var weight = num(fx.vx);
      st.bins[idx] = { amount: num(fx.amount) || 0, weight: weight != null && weight > 0 ? weight : 1 };
      var cad = num(fx.vy);
      if (cad != null && cad > 0) st.cadence = cad;
      if (idx + 1 > st.binCount) st.binCount = idx + 1;
      return { kind: "spectrum-bin", flash: "none" };
    }
    if (name === "doom-at") {
      st.doomAt = num(fx.amount) || 0;
      st.ghostMax = num(fx.vx) || 0;
      if (st.ghostHP == null) st.ghostHP = st.ghostMax;
      var c2 = num(fx.vy);
      if (c2 != null && c2 > 0) st.cadence = c2;
      if (st.doomStart == null) st.doomStart = t;
      if (!st.frozen) {
        st.frozenTrack = st.track.slice();
        st.frozen = true;
        st.tracking = false;
        st.introT0 = t;
      }
      return { kind: "doom-at", flash: "none" };
    }
    if (name === "probe-hit") return { kind: "probe-hit", flash: "none" };
    if (name === "lock-shot") return noteFire(st, Math.round(num(fx.slot) || 0), num(fx.amount) || 0, now);
    if (name === "lock-armed") {
      var bin = Math.round(num(fx.slot) || 0);
      var amount = num(fx.amount) || 0;
      var nArmed = Math.max(st.bins.length, st.binCount, bin + 1, 1);
      missiles[String(fx.unitId)] = { bin: bin, amount: amount, thick: thicknessForTier(bin, nArmed), owner: null };
      if (st.hot < 0) st.hot = bin;
      return { kind: "lock-armed", flash: "none" };
    }
    if (name === "lock-block") {
      st.breaks += 1;
      return { kind: "sever", flash: "none", wall: (num(fx.vy) || 0) < 0.5 };
    }
    if (name === "ghost-hit") {
      st.hits += 1;
      var remain = num(fx.vx);
      if (remain != null) st.ghostHP = remain;
      markTorn(fx.unitId, now);
      return { kind: "tear", flash: "none" };
    }
    if (name === "ghost-down") {
      st.down = true;
      st.ghostHP = 0;
      return { kind: "down", flash: "none" };
    }
    if (name === "doom") {
      st.doomed = true;
      var d = num(fx.amount);
      if (d != null) st.doomAt = d;
      return { kind: "doom", flash: "none" };
    }
    return { kind: "", flash: "none" };
  }

  function note(fx, nowMs) {
    if (!fx) return { kind: "", flash: "none" };
    var now = nowMs == null ? 0 : nowMs;
    var sim = simNow();
    if (fx.name === "ghost-hit") {
      var owner = owners[String(fx.unitId)];
      var stHit = owner != null ? stateFor(owner) : stateFor(fx.unitId);
      return applyEffect(stHit, fx, now, sim);
    }
    if (fx.name === "lock-armed") {
      var binA = Math.round(num(fx.slot) || 0);
      var amtA = num(fx.amount) || 0;
      missiles[String(fx.unitId)] = { bin: binA, amount: amtA, thick: null };
      return { kind: "lock-armed", flash: "none" };
    }
    if (fx.name === "lock-block") {
      return { kind: "sever", flash: "none", wall: (num(fx.vy) || 0) < 0.5 };
    }
    return applyEffect(stateFor(fx.unitId), fx, now, sim);
  }

  function missileOf(id, ownerId) {
    var rec = missiles[String(id)];
    var st = ownerId != null ? buckets[String(ownerId)] : null;
    var n = 1;
    if (st) n = Math.max(st.bins.length, st.binCount, 1);
    if (rec) {
      var bin = rec.bin || 0;
      return { bin: bin, amount: rec.amount || 0, thick: thicknessForTier(bin, Math.max(n, bin + 1)) };
    }
    if (st && st.pending) return st.pending;
    return { bin: 0, amount: 0, thick: 1 };
  }

  function remain(st, now) {
    if (!st || st.doomStart == null) return null;
    return Math.max(0, (st.doomAt || 0) - (now - st.doomStart));
  }

  function tDaggerText(st, now) {
    var left = remain(st, now);
    if (left == null) return "T† —";
    return "T† " + left.toFixed(1);
  }

  function introDuration(requested) {
    var n = Number(requested);
    if (!Number.isFinite(n) || n <= 0) return REWIND_MAX_SEC;
    return Math.min(REWIND_MAX_SEC, n);
  }

  function rewindFrame(track, elapsed, skipped) {
    var dur = REWIND_MAX_SEC;
    if (skipped || !track || !track.length || elapsed < 0 || elapsed > dur) {
      return { active: false, dur: dur, point: null };
    }
    var u = 1 - elapsed / dur;
    var idx = Math.round(u * (track.length - 1));
    if (idx < 0) idx = 0;
    if (idx > track.length - 1) idx = track.length - 1;
    return { active: true, dur: dur, point: track[idx], index: idx };
  }

  function introActive(st, now) {
    if (!st || st.skipped || st.introT0 == null) return false;
    return now - st.introT0 <= REWIND_MAX_SEC + 1e-9;
  }

  function sampleTrack(st, t, x, y) {
    if (!st || !st.tracking) return;
    var last = st.track[st.track.length - 1];
    if (last && Math.hypot(x - last.x, y - last.y) < 0.75 && t - last.t < 0.05) return;
    st.track.push({ t: t, x: x, y: y });
    if (st.track.length > 240) st.track.shift();
  }

  function fmtAmt(n) {
    if (!Number.isFinite(n)) return "·";
    var r = Math.round(n * 10) / 10;
    return String(r);
  }

  function screenOf(x, y, ctx) {
    var arena = root.arena;
    if (arena && typeof arena.screenPos === "function" && ctx) {
      return arena.screenPos(x, y, ctx.scale, ctx.cx, ctx.cy);
    }
    var scale = ctx && ctx.scale ? ctx.scale : 1;
    var cx = ctx && ctx.cx ? ctx.cx : 0;
    var cy = ctx && ctx.cy ? ctx.cy : 0;
    return [cx + x * scale, cy - y * scale];
  }

  function ballBox(el) {
    return {
      w: parseFloat(el.style.width) || el.offsetWidth || 0,
      h: parseFloat(el.style.height) || el.offsetHeight || 0,
      bx: parseFloat(el.style.left) || 0,
      by: parseFloat(el.style.top) || 0,
    };
  }

  function ensure(parent, cls, tag) {
    var node = parent.querySelector(":scope > ." + cls);
    if (node) return node;
    node = document.createElement(tag || "div");
    node.className = cls;
    parent.appendChild(node);
    return node;
  }

  function renderHistogram(host, st, introOn) {
    if (!host || !st) return;
    host.classList.add("nc-hud");
    host.classList.toggle("is-rewinding", !!introOn);
    var caption = host.querySelector(":scope > .nc-caption");
    if (!caption) {
      caption = document.createElement("span");
      caption.className = "nc-caption";
      host.appendChild(caption);
    }
    caption.textContent = CAPTION;
    var row = host.querySelector(":scope > .nc-bins");
    if (!row) {
      row = document.createElement("div");
      row.className = "nc-bins";
      host.appendChild(row);
    }
    var bins = st.bins || [];
    while (row.children.length > bins.length) row.removeChild(row.lastChild);
    var maxW = 1;
    for (var i = 0; i < bins.length; i++) if (bins[i] && bins[i].weight > maxW) maxW = bins[i].weight;
    for (var j = 0; j < bins.length; j++) {
      var cell = row.children[j];
      if (!cell) {
        cell = document.createElement("div");
        cell.className = "nc-bin";
        cell.appendChild(document.createElement("i"));
        cell.appendChild(document.createElement("em"));
        row.appendChild(cell);
      }
      cell.classList.toggle("is-drawn", j === st.hot);
      var w = bins[j] ? bins[j].weight : 1;
      cell.firstChild.style.height = Math.max(8, Math.round((w / maxW) * 100)) + "%";
      cell.lastChild.textContent = bins[j] ? fmtAmt(bins[j].amount) : "·";
    }
    var meta = host.querySelector(":scope > .nc-meta");
    if (!meta) {
      meta = document.createElement("div");
      meta.className = "nc-meta";
      host.appendChild(meta);
    }
    var cad = st.cadence > 0 ? "节拍 " + st.cadence.toFixed(2) + "s" : "节拍 —";
    meta.textContent = st.pulses > 0 ? cad + " · 脉冲 " + st.pulses : cad;
    var skip = host.querySelector(":scope > .nc-skip");
    if (!skip) {
      skip = document.createElement("button");
      skip.type = "button";
      skip.className = "nc-skip";
      skip.textContent = "跳过倒带";
      skip.addEventListener("click", function () {
        st.skipped = true;
      });
      host.appendChild(skip);
    }
  }

  function pinHud(el, panel, slot) {
    var b = ballBox(el);
    var layer = el.parentElement;
    var lw = layer && layer.clientWidth ? layer.clientWidth : 560;
    var localX = slot === 1 ? Math.max(8, lw - 222) : 12;
    var localY = 12;
    panel.style.transform = "none";
    panel.style.left = localX - (b.bx - b.w / 2) + "px";
    panel.style.top = localY - (b.by - b.h / 2) + "px";
  }

  function paintRewind(el, st, ctx) {
    var svg = el.querySelector(":scope > .nc-rewind");
    var now = simNow();
    var frame = rewindFrame(st.frozenTrack, st.introT0 == null ? 99 : now - st.introT0, st.skipped);
    if (!frame.active || !ctx) {
      if (svg) svg.remove();
      return;
    }
    if (typeof document === "undefined") return;
    if (!svg) {
      svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
      svg.setAttribute("class", "nc-rewind");
      el.appendChild(svg);
    }
    var b = ballBox(el);
    svg.style.left = -(b.bx - b.w / 2) + "px";
    svg.style.top = -(b.by - b.h / 2) + "px";
    svg.setAttribute("width", "4000");
    svg.setAttribute("height", "4000");
    var d = "";
    for (var i = 0; i < st.frozenTrack.length; i++) {
      var p = screenOf(st.frozenTrack[i].x, st.frozenTrack[i].y, ctx);
      d += (i === 0 ? "M" : "L") + p[0].toFixed(1) + " " + p[1].toFixed(1) + " ";
    }
    var head = screenOf(frame.point.x, frame.point.y, ctx);
    while (svg.firstChild) svg.removeChild(svg.firstChild);
    var path = document.createElementNS("http://www.w3.org/2000/svg", "path");
    path.setAttribute("d", d);
    var dot = document.createElementNS("http://www.w3.org/2000/svg", "circle");
    dot.setAttribute("cx", head[0].toFixed(1));
    dot.setAttribute("cy", head[1].toFixed(1));
    dot.setAttribute("r", "3.5");
    svg.appendChild(path);
    svg.appendChild(dot);
  }

  function sampleEnemy(st, u, ctx) {
    if (!st.tracking || !ctx) return;
    var best = null;
    var bestD = Infinity;
    var list = ctx.units || [];
    for (var i = 0; i < list.length; i++) {
      var o = list[i];
      if (!o || o.role !== "fighter" || o.slot === u.slot || o.id === u.id) continue;
      var d = Math.hypot((o.x || 0) - (u.x || 0), (o.y || 0) - (u.y || 0));
      if (d < bestD) {
        best = o;
        bestD = d;
      }
    }
    if (best) sampleTrack(st, simNow(), best.x || 0, best.y || 0);
  }

  if (typeof document !== "undefined" && !root.__ncKeys) {
    root.__ncKeys = true;
    document.addEventListener("keydown", function (ev) {
      if (ev.key !== "Escape" && ev.key !== " " && ev.code !== "Space") return;
      var now = simNow();
      var live = false;
      for (var k in buckets) {
        if (introActive(buckets[k], now)) {
          buckets[k].skipped = true;
          live = true;
        }
      }
      if (live) ev.preventDefault();
    });
  }

  root.NoncausalPresent = {
    CAPTION: CAPTION,
    GHOST_OPACITY: GHOST_OPACITY,
    REWIND_MAX_SEC: REWIND_MAX_SEC,
    FLASH_MERGE_MS: FLASH_MERGE_MS,
    RESIDUAL: RESIDUAL,
    LOCK: LOCK,
    createState: createState,
    stateFor: stateFor,
    reset: reset,
    note: note,
    applyEffect: applyEffect,
    bindResidual: bindResidual,
    missileOf: missileOf,
    thicknessForTier: thicknessForTier,
    introDuration: introDuration,
    rewindFrame: rewindFrame,
    introActive: introActive,
    sampleTrack: sampleTrack,
    tDaggerText: tDaggerText,
    remain: remain,
    torn: torn,
    renderHistogram: renderHistogram,
    simNow: simNow,
  };

  root.lookFX.noncausal = {
    unmount: function (el) {
      if (!el) return;
      var hud = el.querySelector(":scope > .nc-hud");
      if (hud) hud.remove();
      var rw = el.querySelector(":scope > .nc-rewind");
      if (rw) rw.remove();
    },
    tick: function (el, u, ctx) {
      if (!el || !u || typeof document === "undefined") return;
      var st = stateFor(u.id);
      sampleEnemy(st, u, ctx);
      var ready = st.binCount > 0;
      var hud = el.querySelector(":scope > .nc-hud");
      if (!ready) {
        if (hud) hud.remove();
      } else {
        hud = hud || ensure(el, "nc-hud");
        renderHistogram(hud, st, introActive(st, simNow()));
        pinHud(el, hud, u.slot || 0);
      }
      paintRewind(el, st, ctx);
    },
  };

  root.lookFX.residual = {
    unmount: function (el) {
      if (!el) return;
      [".nc-caption", ".nc-doom", ".nc-ghost-hp"].forEach(function (sel) {
        var n = el.querySelector(":scope > " + sel);
        if (n) n.remove();
      });
    },
    tick: function (el, u) {
      if (!el || !u || typeof document === "undefined") return;
      bindResidual(u.id, u.ownerId);
      el.dataset.ghostOpacity = String(GHOST_OPACITY);
      var cap = ensure(el, "nc-caption", "span");
      cap.textContent = CAPTION;
      var st = stateFor(u.ownerId);
      var now = simNow();
      var tag = ensure(el, "nc-doom", "span");
      tag.textContent = tDaggerText(st, now);
      var left = remain(st, now);
      tag.classList.toggle("is-due", st.doomed || (left != null && left <= 0));
      var hp = ensure(el, "nc-ghost-hp", "span");
      var bar = hp.querySelector("i");
      if (!bar) {
        bar = document.createElement("i");
        hp.appendChild(bar);
      }
      var max = st.ghostMax || 1;
      var cur = st.ghostHP == null ? max : Math.max(0, st.ghostHP);
      bar.style.width = Math.max(0, Math.min(100, (cur / max) * 100)) + "%";
      el.classList.toggle("is-torn", torn(u.id, typeof performance !== "undefined" ? performance.now() : 0));
    },
  };

  root.lookFX.lock = {
    unmount: function (el) {
      var core = el && el.querySelector(":scope > .nc-core");
      if (core) core.remove();
    },
    tick: function (el, u) {
      if (!el || !u || typeof document === "undefined") return;
      ensure(el, "nc-core", "i");
      var info = missileOf(u.id, u.ownerId);
      el.style.setProperty("--lock-thick", String(info.thick || 1));
    },
    guide: function (u, ctx) {
      if (!u || u.kind !== LOCK || !ctx || typeof ctx.ensureGuide !== "function") return;
      var ghost = null;
      var list = ctx.units || [];
      for (var i = 0; i < list.length; i++) {
        var o = list[i];
        if (o && o.kind === RESIDUAL && o.ownerId === u.ownerId) {
          ghost = o;
          break;
        }
      }
      if (!ghost) return;
      var a = screenOf(u.x, u.y, ctx);
      var b = screenOf(ghost.x, ghost.y, ctx);
      var g = ctx.ensureGuide("guide-nc-" + u.id, "nc-lock");
      if (typeof ctx.placeSeg === "function") ctx.placeSeg(g, a[0], a[1], b[0], b[1], u.kind);
      if (ctx.seenGuides) ctx.seenGuides.add(g.id);
    },
  };
})();
