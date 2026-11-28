(function () {
  var SHOTS = ['tide-state', 'tide-ebb', 'tide-launch', 'tide-surf', 'tide-shed'];
  if (typeof window !== 'undefined' && window.arena && typeof window.arena.registerShot === 'function') {
    for (var si = 0; si < SHOTS.length; si++) {
      window.arena.registerShot(SHOTS[si], function () {});
    }
  }

  var BAND_COUNT = 18;
  var WIDTH_SCALE = 0.25;
  var FLARE_WINDOW = 0.45;
  var CAUSTIC_SIZE = 160;
  var MAX_CANVAS_DIM = 1800;

  var causticCache = null;
  var unitVis = Object.create(null);

  function visKey(id) {
    return String(id);
  }

  function clamp(v, lo, hi) {
    return v < lo ? lo : v > hi ? hi : v;
  }

  function hash1(n) {
    var x = Math.sin(n * 127.1 + 311.7) * 43758.5453123;
    return x - Math.floor(x);
  }

  function hash2(a, b) {
    var x = Math.sin(a * 127.1 + b * 269.5 + 41.3) * 43758.5453123;
    return x - Math.floor(x);
  }

  function getCausticTile() {
    if (causticCache !== null) return causticCache;
    if (typeof document === 'undefined') return null;
    var tile = document.createElement('canvas');
    tile.width = CAUSTIC_SIZE;
    tile.height = CAUSTIC_SIZE;
    var c = tile.getContext && tile.getContext('2d');
    if (!c || typeof c.createImageData !== 'function' || typeof c.putImageData !== 'function') {
      causticCache = { canvas: tile, size: CAUSTIC_SIZE };
      return causticCache;
    }

    var pts = [];
    for (var i = 0; i < 18; i++) {
      pts.push({
        x: hash1(i * 3.1 + 1.7) * CAUSTIC_SIZE,
        y: hash1(i * 5.7 + 9.3) * CAUSTIC_SIZE,
      });
    }

    var img = c.createImageData(CAUSTIC_SIZE, CAUSTIC_SIZE);
    var data = img.data;
    var half = CAUSTIC_SIZE * 0.5;
    var twoPi = Math.PI * 2 / CAUSTIC_SIZE;

    for (var py = 0; py < CAUSTIC_SIZE; py++) {
      for (var px = 0; px < CAUSTIC_SIZE; px++) {
        // Periodic domain warp keeps tile seamless while curving straight Voronoi edges into liquid ripples
        var wx = px + Math.sin(py * twoPi * 2) * 6.5 + Math.cos((px + py) * twoPi) * 3.0;
        var wy = py + Math.cos(px * twoPi * 2) * 6.5 + Math.sin((px - py) * twoPi) * 3.0;
        var d1 = 1e9;
        var d2 = 1e9;
        for (var k = 0; k < pts.length; k++) {
          var dx = Math.abs(wx - pts[k].x);
          var dy = Math.abs(wy - pts[k].y);
          if (dx > half) dx = Math.abs(CAUSTIC_SIZE - dx);
          if (dy > half) dy = Math.abs(CAUSTIC_SIZE - dy);
          var distSq = dx * dx + dy * dy;
          if (distSq < d1) {
            d2 = d1;
            d1 = distSq;
          } else if (distSq < d2) {
            d2 = distSq;
          }
        }
        var diff = Math.sqrt(d2) - Math.sqrt(d1);
        var edge = clamp(1 - diff / 10.5, 0, 1);
        var v = Math.pow(edge, 2.8);
        var glint = Math.pow(clamp(1 - diff / 4.5, 0, 1), 2.1);
        var alpha = clamp(v * 0.78 + glint * 0.35, 0, 1);
        var idx = (py * CAUSTIC_SIZE + px) * 4;
        data[idx] = Math.round(168 + 87 * glint);
        data[idx + 1] = Math.round(240 + 15 * glint);
        data[idx + 2] = 255;
        data[idx + 3] = Math.round(alpha * 255);
      }
    }

    c.putImageData(img, 0, 0);
    causticCache = { canvas: tile, size: CAUSTIC_SIZE };
    return causticCache;
  }

  function getCausticPattern(c, targetCanvas) {
    if (targetCanvas._tideCausticPat) return targetCanvas._tideCausticPat;
    var tile = getCausticTile();
    if (!tile || typeof c.createPattern !== 'function') return null;
    var pat = c.createPattern(tile.canvas, 'repeat');
    if (pat) targetCanvas._tideCausticPat = pat;
    return pat;
  }

  function prepWalls(walls) {
    var out = [];
    if (!walls || !walls.length) return out;
    for (var i = 0; i < walls.length; i++) {
      var w = walls[i];
      var dx = w.x2 - w.x1;
      var dy = w.y2 - w.y1;
      var len = Math.hypot(dx, dy);
      var cx = (w.x1 + w.x2) * 0.5;
      var cy = (w.y1 + w.y2) * 0.5;
      var ax = len > 1e-6 ? dx / len : 1;
      var ay = len > 1e-6 ? dy / len : 0;
      var r = w.radius || 0;
      out.push({
        x1: w.x1,
        y1: w.y1,
        x2: w.x2,
        y2: w.y2,
        r: r,
        square: !!w.square,
        cx: cx,
        cy: cy,
        ax: ax,
        ay: ay,
        hl: len * 0.5 + (w.square ? r : 0),
        hw: r,
      });
    }
    return out;
  }

  function segIntersectsBox(ax, ay, bx, by, w) {
    var acx = ax - w.cx;
    var acy = ay - w.cy;
    var bcx = bx - w.cx;
    var bcy = by - w.cy;
    var sx = acx * w.ax + acy * w.ay;
    var sy = -acx * w.ay + acy * w.ax;
    var ex = bcx * w.ax + bcy * w.ay;
    var ey = -bcx * w.ay + bcy * w.ax;
    var dx = ex - sx;
    var dy = ey - sy;

    var t0 = 0;
    var t1 = 1;
    var checks = [
      [-dx, sx + w.hl],
      [dx, w.hl - sx],
      [-dy, sy + w.hw],
      [dy, w.hw - sy],
    ];
    for (var i = 0; i < 4; i++) {
      var p = checks[i][0];
      var q = checks[i][1];
      if (Math.abs(p) < 1e-9) {
        if (q < 0) return null;
      } else {
        var t = q / p;
        if (p < 0) {
          if (t > t1) return null;
          if (t > t0) t0 = t;
        } else {
          if (t < t0) return null;
          if (t < t1) t1 = t;
        }
      }
    }
    return t0;
  }

  function segSegDistSq(ax, ay, bx, by, cx, cy, dx, dy) {
    var ux = bx - ax;
    var uy = by - ay;
    var vx = dx - cx;
    var vy = dy - cy;
    var wx = ax - cx;
    var wy = ay - cy;
    var a = ux * ux + uy * uy;
    var b = ux * vx + uy * vy;
    var c = vx * vx + vy * vy;
    var d = ux * wx + uy * wy;
    var e = vx * wx + vy * wy;
    var D = a * c - b * b;
    var sN, sD = D, tN, tD = D;
    if (D < 1e-9) {
      sN = 0;
      sD = 1;
      tN = e;
      tD = c;
    } else {
      sN = b * e - c * d;
      tN = a * e - b * d;
      if (sN < 0) {
        sN = 0;
        tN = e;
        tD = c;
      } else if (sN > sD) {
        sN = sD;
        tN = e + b;
        tD = c;
      }
    }
    if (tN < 0) {
      tN = 0;
      if (-d < 0) sN = 0;
      else if (-d > a) sN = sD;
      else {
        sN = -d;
        sD = a;
      }
    } else if (tN > tD) {
      tN = tD;
      if (-d + b < 0) sN = 0;
      else if (-d + b > a) sN = sD;
      else {
        sN = -d + b;
        sD = a;
      }
    }
    var sc = Math.abs(sN) < 1e-9 ? 0 : sN / sD;
    var tc = Math.abs(tN) < 1e-9 ? 0 : tN / tD;
    var dpx = wx + sc * ux - tc * vx;
    var dpy = wy + sc * uy - tc * vy;
    return dpx * dpx + dpy * dpy;
  }

  function clipBand(ox, oy, dx, dy, maxW, w) {
    var ex = ox + dx * maxW;
    var ey = oy + dy * maxW;
    if (w.square) {
      var hitT = segIntersectsBox(ox, oy, ex, ey, w);
      if (hitT === null) return maxW;
      return clamp(hitT * maxW, 0, maxW);
    }
    var rSq = w.r * w.r;
    if (segSegDistSq(ox, oy, ex, ey, w.x1, w.y1, w.x2, w.y2) > rSq) {
      return maxW;
    }
    var lo = 0;
    var hi = maxW;
    for (var it = 0; it < 10; it++) {
      var mid = (lo + hi) * 0.5;
      var mx = ox + dx * mid;
      var my = oy + dy * mid;
      if (segSegDistSq(ox, oy, mx, my, w.x1, w.y1, w.x2, w.y2) <= rSq) {
        hi = mid;
      } else {
        lo = mid;
      }
    }
    return lo;
  }

  function smoothBands(bands, maxW) {
    var clipped = false;
    for (var i = 0; i < bands.length; i++) {
      if (bands[i] < maxW - 0.5) {
        clipped = true;
        break;
      }
    }
    if (!clipped) return bands;
    var out = new Array(bands.length);
    for (var k = 0; k < bands.length; k++) {
      var prev = bands[Math.max(0, k - 1)];
      var curr = bands[k];
      var next = bands[Math.min(bands.length - 1, k + 1)];
      out[k] = Math.min(curr, prev * 0.22 + curr * 0.56 + next * 0.22);
    }
    return out;
  }

  function wallProfile(ux, uy, fx, fy, radius, maxW, walls) {
    var neg = new Array(BAND_COUNT);
    var pos = new Array(BAND_COUNT);
    for (var i = 0; i < BAND_COUNT; i++) {
      neg[i] = maxW;
      pos[i] = maxW;
    }
    if (!walls || !walls.length || maxW <= 0.1) {
      return { neg: neg, pos: pos };
    }
    var nrx = -fy;
    var nry = fx;
    var prx = fy;
    var pry = -fx;
    for (var b = 0; b < BAND_COUNT; b++) {
      var t = BAND_COUNT > 1 ? -1 + (2 * b) / (BAND_COUNT - 1) : 0;
      var bx = ux + fx * t * radius;
      var by = uy + fy * t * radius;
      var minN = maxW;
      var minP = maxW;
      for (var wi = 0; wi < walls.length; wi++) {
        var w = walls[wi];
        var cn = clipBand(bx + nrx * radius * 0.55, by + nry * radius * 0.55, nrx, nry, minN, w);
        if (cn < minN) minN = cn;
        var cp = clipBand(bx + prx * radius * 0.55, by + pry * radius * 0.55, prx, pry, minP, w);
        if (cp < minP) minP = cp;
      }
      neg[b] = minN;
      pos[b] = minP;
    }
    return {
      neg: smoothBands(neg, maxW),
      pos: smoothBands(pos, maxW),
    };
  }

  function sampleBand(bands, s) {
    var idx = clamp(s, 0, 1) * (bands.length - 1);
    var i0 = Math.floor(idx);
    var i1 = Math.min(bands.length - 1, i0 + 1);
    var f = idx - i0;
    return bands[i0] * (1 - f) + bands[i1] * f;
  }

  function traceClosedSpline(c, pts) {
    var n = pts.length;
    if (n < 3) return;
    var sx = (pts[n - 1].x + pts[0].x) * 0.5;
    var sy = (pts[n - 1].y + pts[0].y) * 0.5;
    c.beginPath();
    c.moveTo(sx, sy);
    for (var i = 0; i < n; i++) {
      var p0 = pts[i];
      var p1 = pts[(i + 1) % n];
      c.quadraticCurveTo(p0.x, p0.y, (p0.x + p1.x) * 0.5, (p0.y + p1.y) * 0.5);
    }
    c.closePath();
  }

  function traceOpenSpline(c, pts) {
    var n = pts.length;
    if (n < 2) return;
    c.beginPath();
    c.moveTo(pts[0].x, pts[0].y);
    if (n === 2) {
      c.lineTo(pts[1].x, pts[1].y);
      return;
    }
    for (var i = 1; i < n - 1; i++) {
      var mx = (pts[i].x + pts[i + 1].x) * 0.5;
      var my = (pts[i].y + pts[i + 1].y) * 0.5;
      c.quadraticCurveTo(pts[i].x, pts[i].y, mx, my);
    }
    c.lineTo(pts[n - 1].x, pts[n - 1].y);
  }

  function paintDriftingCaustics(c, pat, time, bounds, alphaA, alphaB, rotOffset, anchor) {
    if (!pat) return;
    var cx = (bounds.minX + bounds.maxX) * 0.5;
    var cy = (bounds.minY + bounds.maxY) * 0.5;
    var diag = Math.hypot(bounds.maxX - bounds.minX, bounds.maxY - bounds.minY) * 0.75 + 80;

    var layers = [
      { ang: 0.28, sc: 0.55, ox: (time * 24) % CAUSTIC_SIZE, oy: (-time * 16) % CAUSTIC_SIZE, alpha: alphaA, comp: null },
      { ang: -0.42, sc: 0.72, ox: (-time * 18) % CAUSTIC_SIZE, oy: (time * 20) % CAUSTIC_SIZE, alpha: alphaB, comp: 'lighter' },
    ];

    for (var li = 0; li < layers.length; li++) {
      var L = layers[li];
      if (!(L.alpha > 0)) continue;
      c.save();
      if (L.comp) c.globalCompositeOperation = L.comp;
      c.globalAlpha = L.alpha;
      c.fillStyle = pat;
      var halfDiag = diag / L.sc + CAUSTIC_SIZE;
      if (anchor) {
        c.translate(anchor.x, anchor.y);
        c.rotate(L.ang + (rotOffset || 0));
        c.scale(L.sc, L.sc);
        c.translate(L.ox, L.oy);
        c.fillRect(-halfDiag, -halfDiag, halfDiag * 2, halfDiag * 2);
      } else {
        var cosA = Math.cos(L.ang);
        var sinA = Math.sin(L.ang);
        c.rotate(L.ang);
        c.scale(L.sc, L.sc);
        c.translate(L.ox, L.oy);
        var rx = (cx * cosA + cy * sinA) / L.sc - L.ox;
        var ry = (-cx * sinA + cy * cosA) / L.sc - L.oy;
        c.fillRect(rx - halfDiag, ry - halfDiag, halfDiag * 2, halfDiag * 2);
      }
      c.restore();
    }
  }

  function getState(el) {
    if (!el._tide) {
      el._tide = {
        rev: -1,
        segs: [],
        segPool: [],
        segById: Object.create(null),
        visWingW: -1,
        ebb: { active: false },
        surfUntil: 0,
        bursts: [],
        seenFx: Object.create(null),
        lastEffectsRef: null,
        lastTime: -1,
        sandbox: undefined,
      };
    }
    return el._tide;
  }

  function resetState(st, sandbox) {
    st.rev = -1;
    st.segs = [];
    st.segPool = [];
    st.segById = Object.create(null);
    st.visWingW = -1;
    st.ebb = { active: false };
    st.surfUntil = 0;
    st.bursts = [];
    st.seenFx = Object.create(null);
    st.lastEffectsRef = null;
    st.lastTime = -1;
    st.sandbox = sandbox;
  }

  function ingestEffects(st, u, ctx, dt) {
    var time = ctx.time != null ? ctx.time : 0;
    var effects = ctx.effects;
    if (!effects || !effects.length) {
      st.lastEffectsRef = effects || null;
    } else {
      var sameRef = effects === st.lastEffectsRef;
      st.lastEffectsRef = effects;
      for (var i = 0; i < effects.length; i++) {
        var fx = effects[i];
        if (!fx || fx.unitId !== u.id) continue;
        var evName = fx.name || fx.type;
        if (evName === 'tide-state') {
          var d = typeof fx.data === 'string' ? JSON.parse(fx.data) : fx.data;
          if (d && d.rev !== st.rev) {
            if (d.rev < st.rev) {
              st.bursts = [];
              st.seenFx = Object.create(null);
              st.segPool = [];
              st.segById = Object.create(null);
              st.visWingW = -1;
            }
            st.rev = d.rev;
            st.surfUntil = d.surfUntil || 0;
            st.ebb = d.ebb || { active: false };
            st.segs = d.segs || [];

            for (var pi = 0; pi < st.segPool.length; pi++) {
              st.segPool[pi].present = false;
            }
            for (var si = 0; si < st.segs.length; si++) {
              var s = st.segs[si];
              var existing = st.segById[s.id];
              if (existing) {
                existing.nw = s.nw;
                existing.pw = s.pw;
                existing.born = s.born;
                existing.life = s.life;
                existing.flare = s.flare || 0;
                existing.evictedAt = -1;
                existing.present = true;
              } else {
                var entry = {
                  id: s.id,
                  x: s.x,
                  y: s.y,
                  fx: s.fx,
                  fy: s.fy,
                  hl: s.hl,
                  nw: s.nw,
                  pw: s.pw,
                  born: s.born,
                  visBorn: s.born,
                  life: s.life,
                  flare: s.flare || 0,
                  evictedAt: -1,
                  present: true,
                  fade: 1,
                };
                st.segById[s.id] = entry;
                st.segPool.push(entry);
              }
            }
            for (var pe = 0; pe < st.segPool.length; pe++) {
              var ep = st.segPool[pe];
              if (!ep.present && ep.evictedAt < 0) {
                ep.evictedAt = time;
              }
            }
          }
          continue;
        }
        if (evName === 'tide-ebb' || evName === 'tide-launch' || evName === 'tide-surf' || evName === 'tide-shed') {
          if (fx.id == null && sameRef) continue;
          var key = fx.id != null
            ? String(fx.id)
            : evName + ':' + Math.round(fx.x) + ':' + Math.round(fx.y) + ':' + (fx.time != null ? fx.time : time);
          if (st.seenFx[key]) continue;
          st.seenFx[key] = true;
          var dur = evName === 'tide-shed' ? 0.55 : evName === 'tide-launch' ? 0.5 : evName === 'tide-surf' ? 0.45 : 0.42;
          st.bursts.push({
            type: evName,
            x: fx.x,
            y: fx.y,
            vx: fx.vx || 0,
            vy: fx.vy || 0,
            amount: fx.amount || 0,
            born: fx.time != null ? fx.time : time,
            dur: dur,
          });
        }
      }
    }

    var kSmooth = dt > 0 ? 1 - Math.exp(-dt * 14) : 0;
    var alive = [];
    for (var k = 0; k < st.segPool.length; k++) {
      var it = st.segPool[k];
      if (dt > 0) {
        it.visBorn += (it.born - it.visBorn) * kSmooth;
      }
      var age = Math.max(0, time - it.visBorn);
      var ageFade = clamp(1 - age / Math.max(0.05, it.life), 0, 1);
      var evictFade = it.evictedAt >= 0 ? clamp(1 - Math.max(0, time - it.evictedAt) / 0.30, 0, 1) : 1;
      it.fade = ageFade * evictFade;
      if (it.fade > 0.002) {
        alive.push(it);
      } else {
        delete st.segById[it.id];
      }
    }
    alive.sort(function (a, b) { return a.id - b.id; });
    st.segPool = alive;

    var kept = [];
    for (var b = 0; b < st.bursts.length; b++) {
      var br = st.bursts[b];
      if (br.born <= time + 0.05 && time - br.born <= br.dur) {
        kept.push(br);
      }
    }
    st.bursts = kept;
  }

  function groupWakeChains(segs) {
    var chains = [];
    var curr = [];
    for (var i = 0; i < segs.length; i++) {
      var s = segs[i];
      if (curr.length > 0) {
        var p = curr[curr.length - 1];
        var dist = Math.hypot(s.x - p.x, s.y - p.y);
        var align = s.fx * p.fx + s.fy * p.fy;
        if (dist > 62 || align < -0.25) {
          chains.push(curr);
          curr = [];
        }
      }
      curr.push(s);
    }
    if (curr.length > 0) chains.push(curr);
    return chains;
  }

  function makeChainGradient(c, tailPt, headPt, cTail, cMid, cHead) {
    if (Math.hypot(headPt.x - tailPt.x, headPt.y - tailPt.y) <= 8) return cMid;
    var g = c.createLinearGradient(tailPt.x, tailPt.y, headPt.x, headPt.y);
    g.addColorStop(0, cTail);
    g.addColorStop(0.32, cMid);
    g.addColorStop(1, cHead);
    return g;
  }

  function drawWakeStreams(c, u, st, time, baseR, walls, pat) {
    var pool = st.segPool;
    if (!pool || !pool.length) return;
    var chains = groupWakeChains(pool);
    var uSpeed = u ? Math.hypot(u.vx || 0, u.vy || 0) : 0;

    for (var ci = 0; ci < chains.length; ci++) {
      var chain = chains[ci];
      var nodes = [];
      for (var i = 0; i < chain.length; i++) {
        var seg = chain[i];
        if (seg.fade <= 0.002) continue;
        var flareAlpha = seg.flare > 0 ? clamp(1 - Math.max(0, time - seg.flare) / FLARE_WINDOW, 0, 1) : 0;
        var grow = 1 + 0.35 * flareAlpha;
        var widthCurve = Math.pow(seg.fade, 1.15);
        nodes.push({
          id: seg.id,
          x: seg.x,
          y: seg.y,
          fx: seg.fx,
          fy: seg.fy,
          nw: seg.nw * widthCurve * grow,
          pw: seg.pw * widthCurve * grow,
          fade: seg.fade,
          flare: flareAlpha,
          arcS: seg.id * 22,
          isLiveHead: false,
        });
      }
      if (nodes.length === 0) continue;

      // Continuous live head bridge connecting the newest wake node seamlessly up to the orb stern
      if (ci === chains.length - 1 && u && uSpeed >= 45) {
        var fx = u.vx / uSpeed;
        var fy = u.vy / uSpeed;
        var lastN = nodes[nodes.length - 1];
        var gapU = Math.hypot(u.x - lastN.x, u.y - lastN.y);
        var fwdU = (u.x - lastN.x) * fx + (u.y - lastN.y) * fy;
        if (gapU <= 58 && fwdU > -6) {
          var hx = u.x - fx * baseR * 0.28;
          var hy = u.y - fy * baseR * 0.28;
          var hDist = Math.hypot(hx - lastN.x, hy - lastN.y);
          if (hDist > 2.5) {
            var hMaxW = Math.max(0, st.visWingW >= 0 ? st.visWingW : uSpeed * WIDTH_SCALE);
            var hProf = wallProfile(u.x, u.y, fx, fy, baseR, hMaxW, walls);
            var rawN = sampleBand(hProf.neg, 0.5);
            var rawP = sampleBand(hProf.pos, 0.5);
            var wBlend = clamp(hDist / 14, 0, 1);
            nodes.push({
              id: lastN.id + hDist / 22,
              x: hx,
              y: hy,
              fx: fx,
              fy: fy,
              nw: lastN.nw * (1 - wBlend) + rawN * wBlend,
              pw: lastN.pw * (1 - wBlend) + rawP * wBlend,
              fade: clamp((uSpeed - 35) / 55, 0, 1),
              flare: 0,
              arcS: lastN.arcS + hDist,
              isLiveHead: true,
            });
          }
        }
      }

      // Compute smooth chord tangents
      var m = nodes.length;
      for (var ti = 0; ti < m; ti++) {
        var cur = nodes[ti];
        var tx = cur.fx;
        var ty = cur.fy;
        if (m > 1) {
          var pPrev = nodes[Math.max(0, ti - 1)];
          var pNext = nodes[Math.min(m - 1, ti + 1)];
          var dx = pNext.x - pPrev.x;
          var dy = pNext.y - pPrev.y;
          var dl = Math.hypot(dx, dy);
          if (dl > 1e-4) {
            tx = dx / dl;
            ty = dy / dl;
            if (tx * cur.fx + ty * cur.fy < 0) {
              tx = -tx;
              ty = -ty;
            }
          }
        }
        cur.tx = tx;
        cur.ty = ty;
      }

      // Interior 3-tap width smoothing to soften turn/clip transitions
      if (m >= 3) {
        var origNW = new Array(m);
        var origPW = new Array(m);
        for (var wi = 0; wi < m; wi++) {
          origNW[wi] = nodes[wi].nw;
          origPW[wi] = nodes[wi].pw;
        }
        for (var wi2 = 1; wi2 < m - 1; wi2++) {
          nodes[wi2].nw = origNW[wi2 - 1] * 0.2 + origNW[wi2] * 0.6 + origNW[wi2 + 1] * 0.2;
          nodes[wi2].pw = origPW[wi2 - 1] * 0.2 + origPW[wi2] * 0.6 + origPW[wi2 + 1] * 0.2;
        }
      }

      // Continuous receding tail slide so expired segments glide forward & taper to zero
      if (m >= 2 && nodes[0].fade < 0.32) {
        var t0 = nodes[0];
        var t1 = nodes[1];
        var uT = clamp((0.32 - t0.fade) / 0.32, 0, 1);
        var slide = uT * uT * (3 - 2 * uT);
        t0.x = t0.x + (t1.x - t0.x) * slide;
        t0.y = t0.y + (t1.y - t0.y) * slide;
        t0.arcS = t0.arcS + (t1.arcS - t0.arcS) * slide;
        var btx = t0.tx + (t1.tx - t0.tx) * slide;
        var bty = t0.ty + (t1.ty - t0.ty) * slide;
        var btl = Math.hypot(btx, bty) || 1;
        t0.tx = btx / btl;
        t0.ty = bty / btl;
        t0.nw *= (1 - slide);
        t0.pw *= (1 - slide);
      }

      // Catmull-Rom 2x spine subdivision for silky curved turns
      var dense = [];
      for (var si2 = 0; si2 < m; si2++) {
        var p1 = nodes[si2];
        dense.push(p1);
        if (si2 < m - 1) {
          var p2 = nodes[si2 + 1];
          if (Math.hypot(p2.x - p1.x, p2.y - p1.y) > 10) {
            var p0 = nodes[Math.max(0, si2 - 1)];
            var p3 = nodes[Math.min(m - 1, si2 + 2)];
            var mtx = p1.tx + p2.tx;
            var mty = p1.ty + p2.ty;
            var mtl = Math.hypot(mtx, mty) || 1;
            dense.push({
              id: (p1.id + p2.id) * 0.5,
              x: 0.5625 * (p1.x + p2.x) - 0.0625 * (p0.x + p3.x),
              y: 0.5625 * (p1.y + p2.y) - 0.0625 * (p0.y + p3.y),
              tx: mtx / mtl,
              ty: mty / mtl,
              nw: (p1.nw + p2.nw) * 0.5,
              pw: (p1.pw + p2.pw) * 0.5,
              fade: (p1.fade + p2.fade) * 0.5,
              flare: (p1.flare + p2.flare) * 0.5,
              arcS: (p1.arcS + p2.arcS) * 0.5,
              isLiveHead: p2.isLiveHead,
              isSubdiv: true,
            });
          }
        }
      }

      var minX = 1e9, minY = 1e9, maxX = -1e9, maxY = -1e9;
      var avgFade = 0;
      var maxFlare = 0;
      for (var di = 0; di < dense.length; di++) {
        var nd = dense[di];
        nd.nrx = -nd.ty;
        nd.nry = nd.tx;
        // Scale wave ripple proportionally to width so receding tails taper smoothly to zero
        nd.nwR = Math.max(0, nd.nw * (1 + 0.065 * Math.sin(nd.arcS * 0.065 - time * 5.2)));
        nd.pwR = Math.max(0, nd.pw * (1 + 0.065 * Math.cos(nd.arcS * 0.062 - time * 4.8)));
        avgFade += nd.fade;
        if (nd.flare > maxFlare) maxFlare = nd.flare;
        var pad = Math.max(nd.nwR, nd.pwR) + 12;
        if (nd.x - pad < minX) minX = nd.x - pad;
        if (nd.x + pad > maxX) maxX = nd.x + pad;
        if (nd.y - pad < minY) minY = nd.y - pad;
        if (nd.y + pad > maxY) maxY = nd.y + pad;
      }
      avgFade /= dense.length;

      var ring = [];
      var coreRing = [];
      var posStream = [];
      var negStream = [];
      var tail = dense[0];
      var head = dense[dense.length - 1];

      // Head rounded cap
      var headCapR = Math.min(head.nwR, head.pwR) * 0.62;
      ring.push({ x: head.x + head.nrx * head.nwR * 0.78 + head.tx * headCapR * 0.35, y: head.y + head.nry * head.nwR * 0.78 + head.ty * headCapR * 0.35 });
      ring.push({ x: head.x + head.tx * headCapR, y: head.y + head.ty * headCapR });
      ring.push({ x: head.x - head.nrx * head.pwR * 0.78 + head.tx * headCapR * 0.35, y: head.y - head.nry * head.pwR * 0.78 + head.ty * headCapR * 0.35 });

      // Positive side from head -> tail
      for (var p = dense.length - 1; p >= 0; p--) {
        var np = dense[p];
        ring.push({ x: np.x - np.nrx * np.pwR, y: np.y - np.nry * np.pwR });
        coreRing.push({ x: np.x - np.nrx * np.pwR * 0.48, y: np.y - np.nry * np.pwR * 0.48 });
        posStream.push({
          x: np.x - np.nrx * np.pwR * (0.42 + 0.11 * Math.sin(np.arcS * 0.065 - time * 5.2)),
          y: np.y - np.nry * np.pwR * (0.42 + 0.11 * Math.sin(np.arcS * 0.065 - time * 5.2)),
        });
      }

      // Tail smooth tip (collapses cleanly to point as tail.nwR/pwR approach 0)
      var tailCapR = Math.min(tail.nwR, tail.pwR) * 0.5;
      ring.push({ x: tail.x - tail.nrx * tail.pwR * 0.65 - tail.tx * tailCapR * 0.35, y: tail.y - tail.nry * tail.pwR * 0.65 - tail.ty * tailCapR * 0.35 });
      ring.push({ x: tail.x - tail.tx * tailCapR, y: tail.y - tail.ty * tailCapR });
      ring.push({ x: tail.x + tail.nrx * tail.nwR * 0.65 - tail.tx * tailCapR * 0.35, y: tail.y + tail.nry * tail.nwR * 0.65 - tail.ty * tailCapR * 0.35 });

      // Negative side from tail -> head
      for (var q = 0; q < dense.length; q++) {
        var nq = dense[q];
        ring.push({ x: nq.x + nq.nrx * nq.nwR, y: nq.y + nq.nry * nq.nwR });
        negStream.push({
          x: nq.x + nq.nrx * nq.nwR * (0.42 + 0.11 * Math.cos(nq.arcS * 0.062 - time * 5.0)),
          y: nq.y + nq.nry * nq.nwR * (0.42 + 0.11 * Math.cos(nq.arcS * 0.062 - time * 5.0)),
        });
      }
      for (var cIdx = 0; cIdx < dense.length; cIdx++) {
        var nc = dense[cIdx];
        coreRing.push({ x: nc.x + nc.nrx * nc.nwR * 0.48, y: nc.y + nc.nry * nc.nwR * 0.48 });
      }

      var midNode = dense[Math.floor(dense.length * 0.5)];
      var tailScale = tail.fade < 0.45 ? 0 : clamp((tail.fade - 0.45) / 0.55, 0, 1) * tail.fade;
      var midAlphaFill = clamp(0.48 * midNode.fade + 0.10 * maxFlare, 0, 0.72);
      var headAlphaFill = clamp(0.52 * head.fade + 0.14 * maxFlare, 0, 0.78);

      // Outer ocean fill with smooth tail-to-head alpha gradient
      c.save();
      traceClosedSpline(c, ring);
      c.fillStyle = makeChainGradient(
        c,
        tail,
        head,
        'rgba(10, 88, 124, ' + (0.42 * tailScale).toFixed(3) + ')',
        'rgba(10, 88, 124, ' + midAlphaFill.toFixed(3) + ')',
        'rgba(12, 98, 136, ' + headAlphaFill.toFixed(3) + ')'
      );
      c.fill();

      // Clip interior for deep core, caustics, and streamlines
      c.clip();

      if (coreRing.length >= 3) {
        traceClosedSpline(c, coreRing);
        c.fillStyle = makeChainGradient(
          c,
          tail,
          head,
          'rgba(4, 34, 62, ' + (0.36 * tailScale).toFixed(3) + ')',
          'rgba(4, 34, 62, ' + (0.44 * midNode.fade).toFixed(3) + ')',
          'rgba(4, 34, 62, ' + (0.48 * head.fade).toFixed(3) + ')'
        );
        c.fill();
      }

      paintDriftingCaustics(
        c,
        pat,
        time,
        { minX: minX, minY: minY, maxX: maxX, maxY: maxY },
        0.30 * avgFade + 0.12 * maxFlare,
        0.22 * avgFade + 0.14 * maxFlare,
        0,
        null
      );

      // Internal curved flow streamlines with tail-to-head fade
      c.lineCap = 'round';
      c.lineJoin = 'round';
      var streamStroke = makeChainGradient(
        c,
        tail,
        head,
        'rgba(155, 244, 255, ' + (0.28 * tailScale).toFixed(3) + ')',
        'rgba(155, 244, 255, ' + (0.32 * midNode.fade + 0.20 * maxFlare).toFixed(3) + ')',
        'rgba(175, 248, 255, ' + (0.36 * head.fade + 0.25 * maxFlare).toFixed(3) + ')'
      );
      if (posStream.length >= 2) {
        traceOpenSpline(c, posStream);
        c.strokeStyle = streamStroke;
        c.lineWidth = 1.4;
        c.stroke();
      }
      if (negStream.length >= 2) {
        traceOpenSpline(c, negStream);
        c.strokeStyle = streamStroke;
        c.lineWidth = 1.4;
        c.stroke();
      }
      c.restore();

      // Glossy shoreline meniscus with smooth tail dissolve
      c.save();
      traceClosedSpline(c, ring);
      var edgeTailA = (0.52 * tailScale).toFixed(3);
      var edgeMidA = clamp(0.58 * midNode.fade + 0.28 * maxFlare, 0, 0.92).toFixed(3);
      var edgeHeadA = clamp(0.65 * head.fade + 0.35 * maxFlare, 0, 0.95).toFixed(3);
      var edgeRGB = maxFlare > 0.1 ? '218, 255, 255' : '130, 236, 252';
      c.strokeStyle = makeChainGradient(
        c,
        tail,
        head,
        'rgba(' + edgeRGB + ', ' + edgeTailA + ')',
        'rgba(' + edgeRGB + ', ' + edgeMidA + ')',
        'rgba(' + edgeRGB + ', ' + edgeHeadA + ')'
      );
      c.lineWidth = 1.45 + 0.7 * maxFlare;
      c.stroke();
      c.restore();

      // Stable seg.id-keyed lacy foam droplets & flare crests (never flips sides on tail prune)
      c.save();
      for (var f = 0; f < nodes.length; f++) {
        var nf = nodes[f];
        if (nf.isLiveHead || nf.fade < 0.06) continue;
        var intId = Math.round(nf.id);
        var sideSign = (intId & 1) === 0 ? 1 : -1;
        var wSide = sideSign > 0 ? nf.nw : nf.pw;
        var bFrac = 0.68 + 0.24 * hash1(intId * 3.7);
        var bx = nf.x + sideSign * (-nf.ty) * wSide * bFrac;
        var by = nf.y + sideSign * nf.tx * wSide * bFrac;
        var fadeCurve = Math.pow(nf.fade, 1.3);
        var br = (1.1 + 1.4 * hash1(intId * 7.1) + 1.2 * nf.flare) * fadeCurve;
        if (br < 0.15) continue;
        c.beginPath();
        c.arc(bx, by, br, 0, Math.PI * 2);
        c.fillStyle = 'rgba(232, 255, 255, ' + clamp(0.45 * fadeCurve + 0.4 * nf.flare, 0, 0.9).toFixed(3) + ')';
        c.fill();

        if (nf.flare > 0.08) {
          c.beginPath();
          c.arc(nf.x, nf.y, Math.min(nf.nw, nf.pw) * 0.42, 0, Math.PI * 2);
          c.strokeStyle = 'rgba(220, 255, 255, ' + (0.65 * nf.flare).toFixed(3) + ')';
          c.lineWidth = 1.6;
          c.stroke();
        }
      }
      c.restore();
    }
  }

  function drawBowWings(c, u, time, radius, walls, surfing, pat, wingW) {
    var speed = Math.hypot(u.vx || 0, u.vy || 0);
    if (speed < 15) return;
    var fx = u.vx / speed;
    var fy = u.vy / speed;
    var nrx = -fy;
    var nry = fx;
    var prx = fy;
    var pry = -fx;

    var maxW = wingW != null && wingW >= 0 ? wingW : speed * WIDTH_SCALE;
    if (maxW < 1) return;
    var prof = wallProfile(u.x, u.y, fx, fy, radius, maxW, walls);
    var intensity = clamp((maxW / WIDTH_SCALE - 15) / 260, 0.15, 1);

    var samples = 18;
    var bowShift = radius * 0.28 + radius * 0.18 * intensity;
    var outerNeg = [];
    var outerPos = [];
    var innerNeg = [];
    var innerPos = [];
    var minX = u.x - maxW - radius * 2;
    var maxX = u.x + maxW + radius * 2;
    var minY = u.y - maxW - radius * 2;
    var maxY = u.y + maxW + radius * 2;

    for (var i = 0; i < samples; i++) {
      var s = i / (samples - 1);
      // s=0 is front bow collar -> s=1 is trailing swept wing tip
      var lx = radius * 1.02 - s * radius * 2.48;
      var env = Math.sin(Math.pow(s, 0.72) * Math.PI);
      var rippleN = 1 + 0.07 * Math.sin(s * 11 - time * 9.5);
      var rippleP = 1 + 0.07 * Math.cos(s * 10.5 - time * 9.0);
      var bandS = clamp(1 - s * 0.95, 0, 1);

      var wn = sampleBand(prof.neg, bandS) * (0.12 + 0.88 * env) * rippleN;
      var wp = sampleBand(prof.pos, bandS) * (0.12 + 0.88 * env) * rippleP;

      var innerBase = radius * (0.62 - 0.18 * s);
      var distOutN = innerBase + wn * 0.92;
      var distOutP = innerBase + wp * 0.92;
      var distInN = Math.max(radius * 0.25, innerBase - radius * 0.12);
      var distInP = Math.max(radius * 0.25, innerBase - radius * 0.12);

      outerNeg.push({ x: u.x + fx * (lx + bowShift * (1 - s)) + nrx * distOutN, y: u.y + fy * (lx + bowShift * (1 - s)) + nry * distOutN });
      outerPos.push({ x: u.x + fx * (lx + bowShift * (1 - s)) + prx * distOutP, y: u.y + fy * (lx + bowShift * (1 - s)) + pry * distOutP });
      innerNeg.push({ x: u.x + fx * lx + nrx * distInN, y: u.y + fy * lx + nry * distInN });
      innerPos.push({ x: u.x + fx * lx + prx * distInP, y: u.y + fy * lx + pry * distInP });
    }

    // Unified seamless teardrop/crescent envelope: bow collar -> outerNeg (front->back) -> neg tip
    // -> smooth stern tuck behind orb center -> pos tip -> outerPos (back->front)
    var pts = [];
    pts.push({ x: u.x + fx * (radius * 1.18 + bowShift * 0.7), y: u.y + fy * (radius * 1.18 + bowShift * 0.7) });
    for (var k = 0; k < samples; k++) pts.push(outerNeg[k]);
    pts.push({ x: u.x - fx * (radius * 1.38) + nrx * (radius * 0.52), y: u.y - fy * (radius * 1.38) + nry * (radius * 0.52) });
    pts.push({ x: u.x - fx * (radius * 0.92), y: u.y - fy * (radius * 0.92) });
    pts.push({ x: u.x - fx * (radius * 1.38) + prx * (radius * 0.52), y: u.y - fy * (radius * 1.38) + pry * (radius * 0.52) });
    for (var k4 = samples - 1; k4 >= 0; k4--) pts.push(outerPos[k4]);

    c.save();
    traceClosedSpline(c, pts);
    var grad = c.createRadialGradient(
      u.x + fx * radius * 0.35,
      u.y + fy * radius * 0.35,
      radius * 0.3,
      u.x,
      u.y,
      radius + maxW * 1.05
    );
    grad.addColorStop(0, surfing ? 'rgba(42, 198, 224, 0.78)' : 'rgba(18, 134, 168, 0.68)');
    grad.addColorStop(0.55, surfing ? 'rgba(14, 128, 162, 0.66)' : 'rgba(9, 82, 118, 0.60)');
    grad.addColorStop(1, 'rgba(6, 48, 82, 0.44)');
    c.fillStyle = grad;
    c.fill();

    // Clip to wing envelope for caustics, streamlines, and internal seafoam
    c.clip();
    paintDriftingCaustics(
      c,
      pat,
      time,
      { minX: minX, minY: minY, maxX: maxX, maxY: maxY },
      0.34 * intensity + (surfing ? 0.14 : 0),
      0.26 * intensity + (surfing ? 0.16 : 0),
      0,
      null
    );

    // Sweeping current ribbons inside each wing
    c.lineCap = 'round';
    for (var side = -1; side <= 1; side += 2) {
      var oArr = side < 0 ? outerNeg : outerPos;
      var iArr = side < 0 ? innerNeg : innerPos;
      for (var rIdx = 0; rIdx < 2; rIdx++) {
        var mixF = rIdx === 0 ? 0.45 : 0.72;
        var rPts = [];
        for (var j = 1; j < samples - 1; j++) {
          rPts.push({
            x: iArr[j].x + (oArr[j].x - iArr[j].x) * mixF,
            y: iArr[j].y + (oArr[j].y - iArr[j].y) * mixF,
          });
        }
        traceOpenSpline(c, rPts);
        c.strokeStyle = surfing
          ? 'rgba(210, 255, 255, ' + (rIdx === 0 ? 0.48 : 0.62) + ')'
          : 'rgba(160, 245, 255, ' + (rIdx === 0 ? 0.28 : 0.44) + ')';
        c.lineWidth = rIdx === 0 ? 1.2 : 1.7;
        c.stroke();
      }
    }
    c.restore();

    // Crisp breaking whitewater rim along leading outer contour
    c.save();
    traceClosedSpline(c, pts);
    c.strokeStyle = surfing
      ? 'rgba(232, 255, 255, 0.92)'
      : 'rgba(184, 249, 255, ' + (0.55 + 0.35 * intensity).toFixed(3) + ')';
    c.lineWidth = surfing ? 2.4 : 1.7;
    c.stroke();

    // Bow breaking crest arc & seafoam spray droplets
    var dropCount = surfing ? 12 : 7;
    for (var d = 0; d < dropCount; d++) {
      var sideD = d % 2 === 0 ? 1 : -1;
      var wD = sideD < 0 ? sampleBand(prof.neg, 0.35 + 0.4 * hash1(d * 2.3)) : sampleBand(prof.pos, 0.35 + 0.4 * hash1(d * 2.3));
      if (wD < 4) continue;
      var phase = (time * 3.2 + hash1(d * 4.9)) % 1;
      var along = radius * (0.6 - phase * 2.2);
      var lat = radius * 0.72 + wD * (0.72 + 0.32 * phase);
      var dxPos = u.x + fx * along + (sideD < 0 ? nrx : prx) * lat;
      var dyPos = u.y + fy * along + (sideD < 0 ? nry : pry) * lat;
      var dRad = (1.1 + 1.5 * hash1(d * 8.1)) * (1 - 0.55 * phase);
      c.beginPath();
      c.arc(dxPos, dyPos, dRad, 0, Math.PI * 2);
      c.fillStyle = 'rgba(236, 255, 255, ' + ((0.75 - 0.55 * phase) * intensity).toFixed(3) + ')';
      c.fill();
    }
    c.restore();
  }

  function drawEbbWhirlpool(c, u, ebb, time, pat) {
    if (!ebb || !ebb.active) return;
    var ebbR = ebb.r || 110;
    var dur = Math.max(0.05, (ebb.until || 0) - (ebb.start || 0));
    var p = clamp((time - (ebb.start || 0)) / dur, 0, 1);
    var fade = p < 0.14 ? p / 0.14 : p > 0.8 ? (1 - p) / 0.2 : 1;
    var rNow = ebbR * (1.03 - 0.14 * p);

    var nPts = 28;
    var ring = [];
    for (var i = 0; i < nPts; i++) {
      var ang = (i / nPts) * Math.PI * 2;
      var wobble = 0.94 + 0.06 * Math.sin(ang * 5 - time * 8.5);
      var rr = rNow * wobble;
      ring.push({ x: u.x + Math.cos(ang) * rr, y: u.y + Math.sin(ang) * rr });
    }

    c.save();
    traceClosedSpline(c, ring);
    var grad = c.createRadialGradient(u.x, u.y, 4, u.x, u.y, rNow);
    grad.addColorStop(0, 'rgba(2, 22, 42, ' + (0.76 * fade).toFixed(3) + ')');
    grad.addColorStop(0.45, 'rgba(9, 76, 114, ' + (0.52 * fade).toFixed(3) + ')');
    grad.addColorStop(0.85, 'rgba(28, 152, 192, ' + (0.28 * fade).toFixed(3) + ')');
    grad.addColorStop(1, 'rgba(64, 212, 236, 0)');
    c.fillStyle = grad;
    c.fill();

    c.clip();
    paintDriftingCaustics(
      c,
      pat,
      time * 1.4,
      { minX: u.x - rNow, minY: u.y - rNow, maxX: u.x + rNow, maxY: u.y + rNow },
      0.34 * fade,
      0.24 * fade,
      -time * 1.2,
      { x: u.x, y: u.y }
    );

    // 10 crisp filled tapered spiral foam ribbons (never dashed lines)
    var arms = 10;
    for (var a = 0; a < arms; a++) {
      var baseAng = (a / arms) * Math.PI * 2 - time * 3.6;
      var outerPts = [];
      var innerPts = [];
      var steps = 10;
      for (var s = 0; s < steps; s++) {
        var frac = s / (steps - 1);
        var rad = rNow * (0.92 - 0.68 * frac);
        var th = baseAng + frac * 1.35;
        var nx = Math.cos(th);
        var ny = Math.sin(th);
        var halfThick = Math.sin(frac * Math.PI) * (2.6 + (a % 2) * 0.9);
        outerPts.push({ x: u.x + nx * (rad + halfThick), y: u.y + ny * (rad + halfThick) });
        innerPts.push({ x: u.x + nx * (rad - halfThick), y: u.y + ny * (rad - halfThick) });
      }
      var ribbon = outerPts.concat(innerPts.reverse());
      traceClosedSpline(c, ribbon);
      c.fillStyle = a % 2 === 0
        ? 'rgba(185, 250, 255, ' + (0.48 * fade).toFixed(3) + ')'
        : 'rgba(120, 224, 248, ' + (0.36 * fade).toFixed(3) + ')';
      c.fill();
    }
    c.restore();

    // Outer suction edge meniscus
    c.save();
    traceClosedSpline(c, ring);
    c.strokeStyle = 'rgba(126, 234, 255, ' + (0.58 * fade).toFixed(3) + ')';
    c.lineWidth = 1.6;
    c.stroke();
    c.restore();
  }

  function drawBursts(c, bursts, time) {
    for (var bi = 0; bi < bursts.length; bi++) {
      var b = bursts[bi];
      var t = clamp((time - b.born) / b.dur, 0, 1);
      if (t <= 0 || t >= 1) continue;
      var fade = Math.pow(1 - t, 1.35);

      if (b.type === 'tide-shed') {
        var maxR = b.vx || 80;
        for (var ringIdx = 0; ringIdx < 2; ringIdx++) {
          var rt = clamp(t * (1 + ringIdx * 0.25) - ringIdx * 0.08, 0, 1);
          if (rt <= 0 || rt >= 1) continue;
          var rRing = maxR * (0.28 + 0.72 * Math.pow(rt, 0.72));
          var pts = [];
          var n = 24;
          for (var i = 0; i < n; i++) {
            var a = (i / n) * Math.PI * 2;
            var wv = 1 + 0.09 * Math.sin(a * 6 + ringIdx * 1.9 - time * 10);
            pts.push({ x: b.x + Math.cos(a) * rRing * wv, y: b.y + Math.sin(a) * rRing * wv });
          }
          c.save();
          traceClosedSpline(c, pts);
          c.strokeStyle = ringIdx === 0
            ? 'rgba(225, 255, 255, ' + (0.82 * fade).toFixed(3) + ')'
            : 'rgba(128, 238, 255, ' + (0.62 * fade).toFixed(3) + ')';
          c.lineWidth = ringIdx === 0 ? 2.4 : 1.7;
          c.stroke();
          c.restore();
        }
        // 14 splash droplets
        for (var d = 0; d < 14; d++) {
          var da = (d / 14) * Math.PI * 2 + hash1(d * 3.1) * 0.3;
          var dist = maxR * (0.25 + 0.85 * Math.pow(t, 0.65)) * (0.78 + 0.32 * hash1(d * 5.7));
          var dr = (1.6 + 1.8 * hash1(d * 8.3)) * (1 - 0.6 * t);
          c.beginPath();
          c.arc(b.x + Math.cos(da) * dist, b.y + Math.sin(da) * dist, dr, 0, Math.PI * 2);
          c.fillStyle = 'rgba(235, 255, 255, ' + (0.85 * fade).toFixed(3) + ')';
          c.fill();
        }
      } else if (b.type === 'tide-launch' || b.type === 'tide-surf') {
        var dirLen = Math.hypot(b.vx, b.vy) || 1;
        var dx = b.vx / dirLen;
        var dy = b.vy / dirLen;
        var px = -dy;
        var py = dx;
        var span = b.type === 'tide-launch' ? 46 : 36;
        var rExp = span * (0.35 + 0.85 * Math.pow(t, 0.7));

        c.save();
        for (var arcIdx = 0; arcIdx < 3; arcIdx++) {
          var arcR = rExp * (0.55 + arcIdx * 0.25);
          var arcPts = [];
          for (var j = -4; j <= 4; j++) {
            var frac = j / 4;
            var ang = frac * 1.15;
            var fwd = Math.cos(ang) * arcR - arcIdx * 5 * t;
            var lat = Math.sin(ang) * arcR * 1.15;
            arcPts.push({ x: b.x + dx * fwd + px * lat, y: b.y + dy * fwd + py * lat });
          }
          traceOpenSpline(c, arcPts);
          c.lineCap = 'round';
          c.strokeStyle = arcIdx === 2
            ? 'rgba(232, 255, 255, ' + (0.85 * fade).toFixed(3) + ')'
            : 'rgba(134, 240, 255, ' + (0.65 * fade).toFixed(3) + ')';
          c.lineWidth = arcIdx === 2 ? 2.4 : 1.7;
          c.stroke();
        }
        // Spray droplets
        for (var k = 0; k < 10; k++) {
          var spread = (hash1(k * 4.3) - 0.5) * 1.8;
          var distL = rExp * (0.5 + 0.7 * hash1(k * 2.9));
          var lx = b.x + (dx * Math.cos(spread) + px * Math.sin(spread)) * distL;
          var ly = b.y + (dy * Math.cos(spread) + py * Math.sin(spread)) * distL;
          c.beginPath();
          c.arc(lx, ly, (1.4 + 1.4 * hash1(k * 9.1)) * (1 - 0.5 * t), 0, Math.PI * 2);
          c.fillStyle = 'rgba(235, 255, 255, ' + (0.82 * fade).toFixed(3) + ')';
          c.fill();
        }
        c.restore();
      } else if (b.type === 'tide-ebb') {
        var er = (b.vx || 95) * (1 - 0.65 * t);
        var ringPts = [];
        for (var m = 0; m < 20; m++) {
          var ma = (m / 20) * Math.PI * 2;
          var mw = 1 + 0.06 * Math.sin(ma * 4 - time * 12);
          ringPts.push({ x: b.x + Math.cos(ma) * er * mw, y: b.y + Math.sin(ma) * er * mw });
        }
        c.save();
        traceClosedSpline(c, ringPts);
        c.strokeStyle = 'rgba(170, 244, 255, ' + (0.72 * fade).toFixed(3) + ')';
        c.lineWidth = 2.0;
        c.stroke();
        c.restore();
      }
    }
  }

  function carveWallsAndPaintFoam(c, walls, u, time) {
    if (!walls || !walls.length) return;
    c.save();
    c.globalCompositeOperation = 'destination-out';
    c.fillStyle = '#000';
    c.strokeStyle = '#000';
    for (var i = 0; i < walls.length; i++) {
      var w = walls[i];
      if (w.square) {
        c.save();
        c.translate(w.cx, w.cy);
        c.rotate(Math.atan2(w.ay, w.ax));
        c.fillRect(-w.hl, -w.hw, w.hl * 2, w.hw * 2);
        c.restore();
      } else {
        c.beginPath();
        c.moveTo(w.x1, w.y1);
        c.lineTo(w.x2, w.y2);
        c.lineCap = 'round';
        c.lineWidth = Math.max(1, w.r * 2);
        c.stroke();
      }
    }
    c.restore();

    // Subtle crisp wall-grazing whitewater flecks near walls close to the unit
    c.save();
    for (var j = 0; j < walls.length; j++) {
      var wl = walls[j];
      var ux = u.x - wl.x1;
      var uy = u.y - wl.y1;
      var wx = wl.x2 - wl.x1;
      var wy = wl.y2 - wl.y1;
      var wlSq = wx * wx + wy * wy || 1;
      var proj = clamp((ux * wx + uy * wy) / wlSq, 0, 1);
      var cx = wl.x1 + wx * proj;
      var cy = wl.y1 + wy * proj;
      var nx = u.x - cx;
      var ny = u.y - cy;
      var dist = Math.hypot(nx, ny);
      if (dist > 140 || dist < 1e-4) continue;
      nx /= dist;
      ny /= dist;
      var tx = -ny;
      var ty = nx;
      var fade = clamp(1 - dist / 140, 0, 1);
      for (var d = 0; d < 5; d++) {
        var off = (d - 2) * 8 + Math.sin(time * 7 + d) * 2.5;
        var outDist = wl.r + 2.2 + 3.5 * hash1(d * 3.7 + j);
        var fx = cx + tx * off + nx * outDist;
        var fy = cy + ty * off + ny * outDist;
        c.beginPath();
        c.arc(fx, fy, 1.3 + 1.1 * hash1(d * 6.1), 0, Math.PI * 2);
        c.fillStyle = 'rgba(225, 255, 255, ' + (0.65 * fade).toFixed(3) + ')';
        c.fill();
      }
    }
    c.restore();
  }

  function ensureBodyArt(el, u, ctx) {
    if (!el || !u) return null;
    var art = el._tideArt;
    if (!art || art.parentNode !== el) {
      art = el.querySelector && el.querySelector('.tide-art');
    }
    if (!art && typeof document !== 'undefined') {
      var clipId = 'tide-orb-clip-' + String(u.id).replace(/[^a-zA-Z0-9_-]/g, '_') + (ctx && ctx.sandbox ? '-s' : '-m');
      art = document.createElement('div');
      art.className = 'tide-art';
      art.innerHTML =
        '<svg viewBox="-32 -32 64 64" aria-hidden="true">' +
        '  <defs>' +
        '    <clipPath id="' + clipId + '"><circle cx="0" cy="0" r="17.2"/></clipPath>' +
        '    <radialGradient id="' + clipId + '-core" cx="38%" cy="34%" r="68%">' +
        '      <stop offset="0%" stop-color="#8cf4ff" stop-opacity="0.34"/>' +
        '      <stop offset="58%" stop-color="#0d6e8f" stop-opacity="0.16"/>' +
        '      <stop offset="100%" stop-color="#031e34" stop-opacity="0.45"/>' +
        '    </radialGradient>' +
        '  </defs>' +
        '  <g clip-path="url(#' + clipId + ')">' +
        '    <circle cx="0" cy="0" r="17.2" fill="url(#' + clipId + '-core)"/>' +
        '    <g class="tide-swirl-a">' +
        '      <path d="M-12,-4 C-6,-13 6,-14 12,-7 C5,-11 -5,-9 -9,-2 Z" fill="#b4ffff" fill-opacity="0.58"/>' +
        '      <path d="M12,4 C6,13 -6,14 -12,7 C-5,11 5,9 9,2 Z" fill="#5fe2f2" fill-opacity="0.52"/>' +
        '    </g>' +
        '    <g class="tide-swirl-b">' +
        '      <path d="M-6,-12 C3,-15 11,-10 13,-1 C9,-8 2,-11 -5,-9 Z" fill="#e6ffff" fill-opacity="0.62"/>' +
        '      <path d="M6,12 C-3,15 -11,10 -13,1 C-9,8 -2,11 5,9 Z" fill="#9ff7ff" fill-opacity="0.48"/>' +
        '    </g>' +
        '    <path d="M-11,-11 C-5,-15 2,-15 7,-11 C2,-13 -5,-12 -9,-8 Z" fill="#ffffff" fill-opacity="0.72"/>' +
        '    <path d="M4,14 C10,12 14,7 15,1 C13,6 9,10 3,12 Z" fill="#7cefff" fill-opacity="0.56"/>' +
        '  </g>' +
        '  <path class="tide-bow" d="M14,-13 C22,-6 22,6 14,13 C19,5 19,-5 14,-13 Z" fill="#e2ffff" fill-opacity="0.88"/>' +
        '  <g class="tide-shield-ring">' +
        '    <path d="M-15,-15 A21.5,21.5 0 0 1 15,-15" fill="none" stroke="#c8ffff" stroke-linecap="round"/>' +
        '    <path d="M15,15 A21.5,21.5 0 0 1 -15,15" fill="none" stroke="#7ae6ff" stroke-linecap="round"/>' +
        '  </g>' +
        '</svg>';
      el.appendChild(art);
    }
    el._tideArt = art;
    el._tideUnitId = u.id;
    return art;
  }

  function tick(el, u, ctx) {
    if (!el || !u) return;
    var art = ensureBodyArt(el, u, ctx);
    if (!art) return;

    var time = ctx && ctx.time != null ? ctx.time : 0;
    var speed = Math.hypot(u.vx || 0, u.vy || 0);
    var ang = speed > 1 ? Math.atan2(-(u.vy || 0), u.vx || 0) : 0;
    var shieldRatio = clamp((speed - 130) / 260, 0, 1);

    art.style.setProperty('--tide-rot-a', (time * 1.45).toFixed(3) + 'rad');
    art.style.setProperty('--tide-rot-b', (-time * 2.1).toFixed(3) + 'rad');
    art.style.setProperty('--tide-ang', ang.toFixed(3) + 'rad');
    art.style.setProperty('--tide-bow-op', clamp((speed - 35) / 240, 0, 0.92).toFixed(3));
    art.style.setProperty('--tide-shield-op', (shieldRatio * 0.92).toFixed(3));
    art.style.setProperty('--tide-shield-scale', (0.94 + 0.08 * shieldRatio).toFixed(3));
    art.style.setProperty('--tide-shield-w', (1.2 + 1.4 * shieldRatio).toFixed(2) + 'px');

    if (el.classList) {
      var st = null;
      var guideEl = ctx && ctx.seenGuides && ctx.seenGuides.get ? ctx.seenGuides.get('tide-floor-' + u.id) : null;
      if (guideEl && guideEl._tide) st = guideEl._tide;
      var surfing = !!(st && st.surfUntil > time);
      var ebbing = !!(st && st.ebb && st.ebb.active);
      var shedding = !!(st && st.bursts && st.bursts.some(function (b) { return b.type === 'tide-shed' && time - b.born < 0.28; }));
      if (!st) {
        var v = unitVis[visKey(u.id)];
        if (v) {
          surfing = v.surfUntil > time;
          ebbing = !!v.ebbActive;
          shedding = v.shedUntil > time;
        }
      }
      el.classList.toggle('tide-surfing', surfing);
      el.classList.toggle('tide-ebbing', ebbing);
      el.classList.toggle('tide-shedding', shedding);
    }
  }

  function guide(u, ctx) {
    if (!u || !ctx || typeof ctx.ensureGuide !== 'function') return;
    var key = 'tide-floor-' + u.id;
    var el = ctx.ensureGuide(key, 'tide-floor');
    if (!el) return;
    if (!el.id) el.id = key;
    if (ctx.seenGuides && typeof ctx.seenGuides.add === 'function') {
      ctx.seenGuides.add(el.id);
    }

    var st = getState(el);
    var time = ctx.time != null ? ctx.time : 0;
    if (st.sandbox !== ctx.sandbox || (st.lastTime >= 0 && time < st.lastTime - 0.05)) {
      resetState(st, ctx.sandbox);
      delete unitVis[visKey(u.id)];
    }
    var dt = st.lastTime >= 0 ? Math.max(0, time - st.lastTime) : 0;
    st.lastTime = time;
    ingestEffects(st, u, ctx, dt);

    var canvas = el._tideCanvas;
    if (!canvas) {
      canvas = el.querySelector && el.querySelector('canvas');
      if (!canvas && typeof document !== 'undefined') {
        canvas = document.createElement('canvas');
        el.appendChild(canvas);
      }
      el._tideCanvas = canvas;
    }
    if (!canvas) return;

    var radius = u.radius || 18;
    var speed = Math.hypot(u.vx || 0, u.vy || 0);
    var targetW = speed * WIDTH_SCALE;
    if (st.visWingW < 0) {
      st.visWingW = targetW;
    } else if (dt > 0) {
      var wRate = targetW >= st.visWingW ? 24 : 14;
      st.visWingW += (targetW - st.visWingW) * (1 - Math.exp(-dt * wRate));
    }

    var scale = ctx.scale || 1;
    var cx = ctx.cx != null ? ctx.cx : 280;
    var cy = ctx.cy != null ? ctx.cy : 280;
    var viewW = Math.max(32, Math.round(cx * 2));
    var viewH = Math.max(32, Math.round(cy * 2));
    var dpr = typeof window !== 'undefined' && window.devicePixelRatio ? clamp(window.devicePixelRatio, 1.5, 2) : 2;
    var cw = Math.max(16, Math.round(viewW * dpr));
    var ch = Math.max(16, Math.round(viewH * dpr));
    var maxDim = Math.max(cw, ch);
    if (maxDim > MAX_CANVAS_DIM) {
      var shrink = MAX_CANVAS_DIM / maxDim;
      cw = Math.max(16, Math.round(cw * shrink));
      ch = Math.max(16, Math.round(ch * shrink));
    }

    var tv = el._tideView;
    if (!tv || tv.viewW !== viewW || tv.viewH !== viewH || tv.cw !== cw || tv.ch !== ch) {
      canvas.width = cw;
      canvas.height = ch;
      el.style.left = '0px';
      el.style.top = '0px';
      el.style.width = viewW + 'px';
      el.style.height = viewH + 'px';
      el._tideView = { viewW: viewW, viewH: viewH, cw: cw, ch: ch };
    }

    var c = canvas.getContext && canvas.getContext('2d');
    if (!c) return;

    c.setTransform(1, 0, 0, 1, 0, 0);
    c.clearRect(0, 0, cw, ch);

    var sxRatio = cw / viewW;
    var syRatio = ch / viewH;
    c.setTransform(scale * sxRatio, 0, 0, -scale * syRatio, cx * sxRatio, cy * syRatio);

    var pat = getCausticPattern(c, canvas);
    var walls = prepWalls(ctx.walls);
    var surfing = st.surfUntil > time;
    var shedUntil = 0;
    for (var bi = 0; bi < st.bursts.length; bi++) {
      if (st.bursts[bi].type === 'tide-shed' && st.bursts[bi].born + 0.28 > shedUntil) {
        shedUntil = st.bursts[bi].born + 0.28;
      }
    }
    unitVis[visKey(u.id)] = {
      surfUntil: st.surfUntil || 0,
      ebbActive: !!(st.ebb && st.ebb.active),
      shedUntil: shedUntil,
    };

    drawWakeStreams(c, u, st, time, radius, walls, pat);
    drawBowWings(c, u, time, radius, walls, surfing, pat, st.visWingW);
    drawEbbWhirlpool(c, u, st.ebb, time, pat);
    drawBursts(c, st.bursts, time);
    carveWallsAndPaintFoam(c, walls, u, time);
  }

  function unmount(el) {
    if (!el) return;
    if (el._tideArt) {
      if (el._tideArt.parentNode) el._tideArt.parentNode.removeChild(el._tideArt);
      el._tideArt = null;
    }
    if (el.classList) {
      el.classList.remove('tide-surfing', 'tide-ebbing', 'tide-shedding');
    }
    if (el._tideUnitId != null) {
      delete unitVis[visKey(el._tideUnitId)];
      el._tideUnitId = null;
    }
  }

  if (typeof window !== 'undefined') {
    window.lookFX = window.lookFX || {};
    window.lookFX.tide = {
      tick: tick,
      guide: guide,
      unmount: unmount,
    };
  }
})();
