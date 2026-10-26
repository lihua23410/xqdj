(function () {
  window.lookFX = window.lookFX || {};
  var S = (window.charcoalFX = window.charcoalFX || {});
  S.frames = S.frames || {};
  S.prev = S.prev || {};
  S.frameAt = S.frameAt || {};
  S.handAt = S.handAt || {};
  S.groups = S.groups || new Map();
  S.ownerSlots = S.ownerSlots || new Map();

  var FIRE_KIND = "炭翁";
  var dirty = false;

  function schedule() {
    if (dirty) return;
    dirty = true;
    if (typeof queueMicrotask === "function") queueMicrotask(flush);
    else Promise.resolve().then(flush);
  }

  function flush() {
    dirty = false;
    try {
      reconcile();
    } catch (e) {
      if (window.console && console.warn) console.warn("charcoal fx:", e);
    }
  }

  function hostOf(id) {
    if (id == null) return null;
    return document.getElementById("u-" + id);
  }

  function boxOf(host) {
    var left = parseFloat(host.style.left);
    var top = parseFloat(host.style.top);
    var w = parseFloat(host.style.width) || 36;
    var h = parseFloat(host.style.height) || 36;
    if (!Number.isFinite(left)) left = 0;
    if (!Number.isFinite(top)) top = 0;
    return [left, top, w, h];
  }

  function hostPoint(host, xPct, yPct) {
    var b = boxOf(host);
    var x = xPct == null ? 50 : xPct;
    var y = yPct == null ? 50 : yPct;
    return [b[0] + (x / 100 - 0.5) * b[2], b[1] + (y / 100 - 0.5) * b[3]];
  }

  // 两只炭翁可能把炭交给同一个目标；给每只主人一点错位，别完全叠住。
  function ownerOffset(owner) {
    var n = S.ownerSlots.get(owner);
    if (n == null) {
      n = S.ownerSlots.size;
      S.ownerSlots.set(owner, n);
    }
    return n * 2.2;
  }

  function startAngle(n) {
    if (n <= 2) return 0;
    if (n === 3) return 90;
    if (n === 4) return 45;
    if (n === 5) return 90;
    return 0;
  }

  function hexSpot(i, n, owner) {
    var count = Math.max(1, n);
    var ring = Math.floor(i / count);
    var a = ((startAngle(count) + (i / count) * 360) * Math.PI) / 180;
    var r = 78 + ownerOffset(owner) * 0.9 + ring * 8;
    return {
      x: 50 + Math.cos(a) * r,
      y: 50 + Math.sin(a) * r,
      s: 20,
      rot: ((i * 47) % 70) - 35,
    };
  }

  function applySpot(dot, spot) {
    dot.style.setProperty("--x", spot.x.toFixed(2) + "%");
    dot.style.setProperty("--y", spot.y.toFixed(2) + "%");
    dot.style.setProperty("--s", spot.s.toFixed(2) + "%");
    dot.style.setProperty("--rot", spot.rot.toFixed(1) + "deg");
  }

  function popDot(dot) {
    if (!dot.animate) return;
    dot.animate(
      [
        { transform: "translate(-50%, -50%) scale(0.15)", opacity: 0 },
        { transform: "translate(-50%, -50%) scale(1.35)", opacity: 1, offset: 0.7 },
        { transform: "translate(-50%, -50%) scale(1)", opacity: 1 },
      ],
      { duration: 420, easing: "cubic-bezier(.2, .9, .2, 1.1)" },
    );
  }

  function overRoot() {
    return document.getElementById("over") || undefined;
  }

  function sparkAtXY(x, y, n, power, root) {
    if (!window.arena || !arena.spawnFx) return;
    for (var i = 0; i < n; i++) {
      var a = Math.random() * Math.PI * 2;
      var d = (5 + Math.random() * 12) * (power || 1);
      arena.spawnFx("charcoal-spark", x, y, FIRE_KIND, {
        "--dx": (Math.cos(a) * d).toFixed(1) + "px",
        "--dy": (Math.sin(a) * d - 5).toFixed(1) + "px",
      }, root);
    }
  }

  function puffAtXY(x, y, n, power, root) {
    if (!window.arena || !arena.spawnFx) return;
    for (var i = 0; i < n; i++) {
      arena.spawnFx("charcoal-puff", x, y, FIRE_KIND, {
        "--dx": ((Math.random() - 0.5) * 22 * (power || 1)).toFixed(1) + "px",
        "--dy": (-10 - Math.random() * 18 * (power || 1)).toFixed(1) + "px",
        "--s": (6 + Math.random() * 8).toFixed(1) + "px",
      }, root);
    }
  }

  function sparkAt(host, xPct, yPct, n, power, root) {
    if (!host) return;
    var p = hostPoint(host, xPct, yPct);
    sparkAtXY(p[0], p[1], n, power, root);
  }

  function forge(host, added) {
    if (!host || !window.arena || !arena.spawnFx) return;
    var p = hostPoint(host, 50, 56);
    if (S.flare) S.flare(host);
    arena.spawnFx("charcoal-forge-core", p[0], p[1], FIRE_KIND);
    arena.spawnFx("charcoal-forge-ring", p[0], p[1], FIRE_KIND);
    sparkAtXY(p[0], p[1], 5 + Math.min(4, added) * 2, 1.2);
    puffAtXY(p[0], p[1] - 4, 2 + Math.min(3, added), 1);
  }

  function land(host, added) {
    if (!host || !window.arena || !arena.spawnFx) return;
    var p = hostPoint(host, 50, 36);
    var root = overRoot();
    arena.spawnFx("charcoal-land", p[0], p[1], FIRE_KIND, {}, root);
    sparkAtXY(p[0], p[1], 3 + Math.min(3, added), 0.85, root);
    puffAtXY(p[0], p[1] - 4, 1, 0.7, root);
  }

  function burn(host, count, isSelf, value) {
    if (!host || !window.arena || !arena.spawnFx) return;
    var p = hostPoint(host, 50, 52);
    var n = Math.max(1, Math.min(6, count));
    // 爆炸跟着炭的面值走；大炭烧完不能只冒个小火星。
    var factor = Math.sqrt(Math.max(1, (value || 6) / 6));
    var r = Math.max(27 + n * 5, 36 * factor).toFixed(0) + "px";
    var root = overRoot();
    arena.spawnFx("charcoal-burn-flash", p[0], p[1], FIRE_KIND, { "--r": r }, root);
    arena.spawnFx("charcoal-burn-ring", p[0], p[1], FIRE_KIND, { "--r": r }, root);
    arena.spawnFx("charcoal-burn-smoke", p[0], p[1] - 4, FIRE_KIND, { "--r": r }, root);
    sparkAtXY(p[0], p[1], 6 + n * 2, 1.35, root);
    for (var i = 0; i < 2 + n; i++) {
      arena.spawnFx("charcoal-ash", p[0], p[1], FIRE_KIND, {
        "--dx": ((Math.random() - 0.5) * 30).toFixed(1) + "px",
        "--dy": (4 + Math.random() * 16).toFixed(1) + "px",
      }, root);
    }
    if (isSelf && S.flare) S.flare(host);
  }

  function countWhite(arr) {
    var n = 0;
    for (var i = 0; i < arr.length; i++) if (arr[i] && arr[i].white) n++;
    return n;
  }

  function maxValue(arr) {
    var m = 0;
    for (var i = 0; i < arr.length; i++) {
      var v = Number(arr[i] && arr[i].value);
      if (Number.isFinite(v) && v > m) m = v;
    }
    return m || 6;
  }

  function ensureGroup(owner, holder, host) {
    var byHolder = S.groups.get(owner);
    if (!byHolder) {
      byHolder = new Map();
      S.groups.set(owner, byHolder);
    }
    var group = byHolder.get(holder);
    if (group && (!group.isConnected || group.parentElement !== host)) {
      group.remove();
      group = null;
    }
    if (!group) {
      group = document.createElement("div");
      group.className = "coal-group";
      group.dataset.owner = String(owner);
      group.dataset.holder = String(holder);
      host.appendChild(group);
      byHolder.set(holder, group);
    }
    group.style.setProperty("--coal-r", ((boxOf(host)[2] || 36) / 2).toFixed(1) + "px");
    return group;
  }

  function removeGroup(owner, holder) {
    var byHolder = S.groups.get(owner);
    if (!byHolder) return;
    var group = byHolder.get(holder);
    if (group) group.remove();
    byHolder.delete(holder);
    if (!byHolder.size) S.groups.delete(owner);
  }

  function removeOwnerGroups(owner) {
    var byHolder = S.groups.get(owner);
    if (!byHolder) return;
    byHolder.forEach(function (group) {
      group.remove();
    });
    S.groups.delete(owner);
  }

  function syncDots(group, host, owner, holder, prevArr, nextArr) {
    var isCharcoal = host.classList.contains("look-charcoal");
    group.classList.toggle("self", isCharcoal);
    group.style.setProperty("--coal-r", ((boxOf(host)[2] || 36) / 2).toFixed(1) + "px");

    while (group.childElementCount > nextArr.length) {
      group.lastElementChild.remove();
    }
    while (group.childElementCount < nextArr.length) {
      var created = document.createElement("i");
      created.className = "coal-dot";
      group.appendChild(created);
      popDot(created);
    }
    var n = nextArr.length;
    for (var i = 0; i < n; i++) {
      var dot = group.children[i];
      var spot = hexSpot(i, n, owner);
      var value = Number(nextArr[i] && nextArr[i].value);
      if (!Number.isFinite(value) || value <= 0) value = 6;
      // 6 点伤害 = 现在的大小（球体直径的 20%）；往上不封顶。
      spot.s = (20 * value) / 6;
      applySpot(dot, spot);
      var white = !!(nextArr[i] && nextArr[i].white);
      var wasWhite = dot.classList.contains("white");
      dot.classList.toggle("white", white);
      if (white && !wasWhite) {
        sparkAt(host, spot.x, spot.y, 2, 0.65);
      }
    }
    group.dataset.hot = countWhite(nextArr) > 0 ? "1" : "0";
  }

  function updateBurden(holder) {
    var host = hostOf(holder);
    if (!host) return;
    var total = 0;
    for (var key in S.frames) {
      var map = S.frames[key];
      if (!map || typeof map.get !== "function") continue;
      var arr = map.get(Number(holder));
      if (arr) total += arr.length;
    }
    var load = Math.min(1, total / 6);
    host.style.setProperty("--coal-load", load.toFixed(3));
    if (host.classList.contains("look-charcoal")) {
      var old = host.querySelector(":scope > .coal-burden");
      if (old) old.remove();
      return;
    }
    var burden = host.querySelector(":scope > .coal-burden");
    if (total <= 0) {
      if (burden) burden.remove();
      return;
    }
    if (!burden) {
      burden = document.createElement("div");
      burden.className = "coal-burden";
      burden.innerHTML =
        '<i class="coal-ground"></i>' +
        '<i class="coal-ash"></i><i class="coal-ash"></i><i class="coal-ash"></i>';
      host.appendChild(burden);
    }
    burden.style.setProperty("--load", load.toFixed(3));
    burden.dataset.load = String(total);
    burden.classList.toggle("full", total >= 6);
    burden.classList.toggle("over", total > 6);
  }

  function liveIds() {
    try {
      var st = (typeof state !== "undefined" && state) || window.state;
      if (st && Array.isArray(st.units)) {
        var set = new Set();
        for (var i = 0; i < st.units.length; i++) set.add(st.units[i].id);
        return set;
      }
    } catch (e) {}
    return null;
  }

  function reconcile() {
    var now = performance.now();
    var live = liveIds();

    for (var key in S.frames) {
      var owner = Number(key);
      var at = S.frameAt[key] || 0;
      if (live && !live.has(owner) && now - at > 400) {
        removeOwnerGroups(owner);
        delete S.frames[key];
        delete S.prev[key];
        delete S.frameAt[key];
        delete S.handAt[key];
      }
    }

    var touched = new Set();
    for (var okey in S.frames) {
      var ownerId = Number(okey);
      var next = S.frames[okey];
      var prev = S.prev[okey] || new Map();
      var handed = now - (S.handAt[okey] || 0) < 170;
      var holders = new Set();
      prev.forEach(function (_, h) {
        holders.add(h);
      });
      next.forEach(function (_, h) {
        holders.add(h);
      });
      holders.forEach(function (holder) {
        touched.add(holder);
        var host = hostOf(holder);
        var prevArr = prev.get(holder) || [];
        var nextArr = next.get(holder) || [];
        if (!host) {
          removeGroup(ownerId, holder);
          return;
        }
        if (!nextArr.length) {
          if (prevArr.length && !handed) burn(host, prevArr.length, holder === ownerId, maxValue(prevArr));
          removeGroup(ownerId, holder);
          return;
        }
        var group = ensureGroup(ownerId, holder, host);
        syncDots(group, host, ownerId, holder, prevArr, nextArr);
        if (nextArr.length > prevArr.length) {
          var added = nextArr.length - prevArr.length;
          if (holder === ownerId) forge(host, added);
          else land(host, added);
        } else if (!handed && countWhite(prevArr) > countWhite(nextArr)) {
          burn(host, countWhite(prevArr) - countWhite(nextArr), holder === ownerId, maxValue(prevArr));
        } else if (!handed && nextArr.length < prevArr.length) {
          burn(host, prevArr.length - nextArr.length, holder === ownerId, maxValue(prevArr));
        }
      });
      S.prev[okey] = next;
    }

    var burdens = new Set(touched);
    var existing = document.querySelectorAll(".coal-burden");
    for (var i = 0; i < existing.length; i++) {
      var parent = existing[i].parentElement;
      if (!parent || !parent.id || parent.id.indexOf("u-") !== 0) continue;
      var id = Number(parent.id.slice(2));
      if (Number.isFinite(id)) burdens.add(id);
    }
    burdens.forEach(updateBurden);
  }

  function animateHand(fx, ctx) {
    if (!fx || !ctx || !window.arena || !arena.spawnFx) return;
    var kind = fx.kind || FIRE_KIND;
    var root = document.getElementById("over") || undefined;
    var el = arena.spawnFx("charcoal-hand", ctx.x, ctx.y, kind, {}, root);
    if (!el) return;
    var handValue = Number(fx.amount);
    if (!Number.isFinite(handValue) || handValue <= 0) handValue = 6;
    // 飞行中的炭跟落点上的炭用同一套尺寸：6 点 = 13px。
    var handSize = (13 * handValue) / 6;
    el.style.setProperty("--s", handSize.toFixed(1) + "px");
    var targetHost = hostOf(fx.slot);
    var fixedTarget =
      window.arena.screenPos && Number.isFinite(ctx.scale)
        ? arena.screenPos(fx.vx || 0, fx.vy || 0, ctx.scale, ctx.cx, ctx.cy)
        : [ctx.x, ctx.y];
    var x0 = ctx.x;
    var y0 = ctx.y;
    var start = performance.now();
    var dist0 = Math.hypot(fixedTarget[0] - x0, fixedTarget[1] - y0);
    var dur = Math.max(260, Math.min(560, 240 + dist0 * 0.55));
    var lastTrail = 0;

    function targetPoint() {
      if (targetHost && targetHost.isConnected) return hostPoint(targetHost, 50, 40);
      return fixedTarget;
    }

    function frame(now) {
      if (!el.isConnected) return;
      var t = Math.min(1, (now - start) / dur);
      var tp = targetPoint();
      var mx = (x0 + tp[0]) / 2;
      var my = (y0 + tp[1]) / 2;
      var dx = tp[0] - x0;
      var dy = tp[1] - y0;
      var dist = Math.hypot(dx, dy) || 1;
      var nx = -dy / dist;
      var ny = dx / dist;
      var bend = 20 + Math.min(30, dist * 0.06);
      var cx = mx + nx * bend * 0.35;
      var cy = my - bend;
      var u = t;
      var iu = 1 - u;
      var x = iu * iu * x0 + 2 * iu * u * cx + u * u * tp[0];
      var y = iu * iu * y0 + 2 * iu * u * cy + u * u * tp[1];
      var vx = 2 * iu * (cx - x0) + 2 * u * (tp[0] - cx);
      var vy = 2 * iu * (cy - y0) + 2 * u * (tp[1] - cy);
      el.style.left = x + "px";
      el.style.top = y + "px";
      el.style.setProperty("--tail-ang", Math.atan2(vy, vx).toFixed(3) + "rad");
      var s = 0.9 + 0.34 * Math.sin(Math.PI * u);
      el.style.transform = "translate(-50%, -50%) scale(" + s.toFixed(3) + ")";
      el.style.opacity = u > 0.86 ? String(1 - (u - 0.86) / 0.14) : "1";
      if (u < 0.92 && now - lastTrail > 42) {
        lastTrail = now;
        arena.spawnFx("charcoal-trail", x, y, kind, {
          "--dx": (-vx * 0.015).toFixed(1) + "px",
          "--dy": (-vy * 0.015).toFixed(1) + "px",
        }, root);
      }
      if (t < 1) {
        requestAnimationFrame(frame);
        return;
      }
      el.remove();
      arena.spawnFx("charcoal-land", tp[0], tp[1], kind, {
        "--r": Math.max(18, handSize * 1.3).toFixed(1) + "px",
      }, root);
      sparkAtXY(tp[0], tp[1], 4, 0.9);
      puffAtXY(tp[0], tp[1] - 4, 1, 0.7);
    }
    requestAnimationFrame(frame);
  }

  arena.registerShot("coals", function (fx) {
    if (!fx || fx.unitId == null) return;
    S.frames[fx.unitId] = new Map();
    S.frameAt[fx.unitId] = performance.now();
    schedule();
  });

  arena.registerShot("coal", function (fx) {
    if (!fx || fx.unitId == null || fx.slot == null) return;
    var owner = fx.unitId;
    var frame = S.frames[owner];
    if (!frame) {
      frame = S.frames[owner] = new Map();
      S.frameAt[owner] = performance.now();
    }
    var holder = fx.slot;
    var arr = frame.get(holder);
    if (!arr) {
      arr = [];
      frame.set(holder, arr);
    }
    var i = Math.max(0, Math.floor(fx.vx || 0));
    // 炭的面值在 VY；6 点 = 基准大小，往上不封顶。
    var value = Number(fx.vy);
    if (!Number.isFinite(value) || value <= 0) value = 6;
    arr[i] = { white: (fx.amount || 0) >= 0.5, value: value };
    for (var k = 0; k < arr.length; k++) {
      if (!arr[k]) arr[k] = { white: false, value: 6 };
    }
    S.frameAt[owner] = performance.now();
    schedule();
  });

  arena.registerShot("hand", function (fx, ctx) {
    if (!fx || !ctx) return;
    S.handAt[fx.unitId] = performance.now();
    var ownerHost = hostOf(fx.unitId);
    if (ownerHost) {
      if (S.throwPose) S.throwPose(ownerHost);
      var p = hostPoint(ownerHost, 50, 56);
      sparkAtXY(p[0], p[1], 4, 0.8);
      puffAtXY(p[0], p[1] - 4, 2, 0.8);
    }
    animateHand(fx, ctx);
  });
})();
