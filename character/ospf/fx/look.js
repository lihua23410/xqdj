// 数据中心皮肤：本体是一台立式机柜，节点是机架里的 1U 交换机，报文是发光
// 水晶头，落点按断路器跳闸画。装机和发报两条进度留在面板灯带上，节点之间
// 用淡色网线连成机房布线图。
(function () {
  const hooks = window.lookFX = window.lookFX || {};
  const NS = 'http://www.w3.org/2000/svg';
  const clocks = new Map(), roles = new Map();
  // 骨干、普通 area、完全 stub、完全 NSSA：四块分区各自的机架标签色。
  const tones = ['#fff3cd', '#67adff', '#939da8', '#7ce7a4'];
  const AREA_NAME = ['骨干', '普通', 'STUB', 'NSSA'];
  const CABLE_CLASS = {
    inner: 'ospf-cable-inner',
    uplink: 'ospf-cable-uplink',
    inject: 'ospf-cable-inject',
    live: 'ospf-cable-live',
  };
  let lastTime = -Infinity;

  function sync(time) {
    if (time < lastTime) {
      clocks.clear();
      roles.clear();
      cuts.clear();
      cables.clear();
      // 重开一局时节点 id 会从头来，地块的"已就位"记录要跟着清掉。
      document.querySelectorAll('.ospf-map').forEach((m) => m._known && m._known.clear());
    }
    lastTime = time;
  }
  function clamp01(v) { return v < 0 ? 0 : v > 1 ? 1 : v; }
  // due 是绝对到期时刻，span 是本项间隔；满格表示这一拍已经走完。
  function cycle(due, time, span) { return clamp01(1 - (due - time) / span); }

  // 坐标按球的实际像素走（1 单位 = 1px），36px 的机柜里只放得下三台 1U 和两条灯带。
  function rackFace() {
    const units = [0, 1, 2].map((i) => {
      const y = 10.6 + i * 5.7;
      return `<g class="ospf-u">
        <rect class="ospf-u-body" x="8.5" y="${y}" width="19" height="4.6" rx="1.2"/>
        <path class="ospf-u-bays" d="M11 ${y + 1.3}v2M13.1 ${y + 1.3}v2"/>
        <circle class="ospf-u-led" cx="25" cy="${y + 2.3}" r="1.15"/>
      </g>`;
    }).join('');
    return `<svg class="ospf-rack" viewBox="0 0 36 36" aria-hidden="true">
      <rect class="ospf-rail" x="1.6" y="1.4" width="32.8" height="33.2" rx="5.5"/>
      <rect class="ospf-shell" x="5" y="4" width="26" height="28" rx="2.6"/>
      <path class="ospf-shell-gloss" d="M7.2 5.6h21.6"/>
      <rect class="ospf-bar-track" x="8.5" y="6.4" width="19" height="2" rx="1"/>
      <rect class="ospf-bar ospf-bar-send" x="8.5" y="6.4" width="19" height="2" rx="1"/>
      ${units}
      <rect class="ospf-bar-track" x="8.5" y="27.6" width="19" height="2" rx="1"/>
      <rect class="ospf-bar ospf-bar-install" x="8.5" y="27.6" width="19" height="2" rx="1"/>
    </svg>`;
  }

  // 机架里的 1U 接入交换机：散热格栅 + 两颗分区灯，20px 内只留这些。
  function switchFace() {
    return `<svg class="ospf-switch" viewBox="0 0 20 20" aria-hidden="true">
      <rect class="ospf-sw-halo" x="0.4" y="4.4" width="19.2" height="11.2" rx="3.2"/>
      <rect class="ospf-sw-frame" x="1" y="6.2" width="18" height="7.6" rx="2"/>
      <path class="ospf-sw-vents" d="M4.6 8.2v3.6M7 8.2v3.6"/>
      <circle class="ospf-sw-led a" cx="13.4" cy="10" r="1.4"/>
      <circle class="ospf-sw-led b" cx="16.4" cy="10" r="1.4"/>
    </svg>`;
  }

  for (const [name, data] of [['ospf-clock', clocks], ['ospf-role', roles]]) {
    window.arena.registerShot(name, (fx, ctx) => {
      sync(ctx.time);
      data.set(fx.unitId, [fx.vx, fx.vy]);
    });
  }
  // 后端在每次发报时把这条报文的实际航线逐段发过来。皮肤的网线以它为准，
  // 不再自己按拓扑猜"报文该走哪条"。
  const cables = new Map();
  window.arena.registerShot('ospf-cable', (fx, ctx) => {
    const q = (v) => Math.round(v / 2) * 2;
    const key = `${q(fx.x)},${q(fx.y)}>${q(fx.vx)},${q(fx.vy)}`;
    cables.set(fx.unitId + ':' + key, [fx.x, fx.y, fx.vx, fx.vy, Date.now()]);
  });

  // 分区几何：x 是场地内切半径，y 是扇区旋转角，用来算绕开骨干的折线。
  const cuts = new Map();
  window.arena.registerShot('ospf-clock', (fx, ctx) => {
    if (fx.x > 0) cuts.set(fx.unitId, [fx.x, fx.y, fx.amount]);
  });
  window.arena.registerShot('ospf-arrival-blast', (fx, ctx) => {
    const el = window.arena.spawnFx('fx-ospf-arrival-blast', ctx.x, ctx.y, ctx.kind, {
      '--ospf-blast-size': `${2 * fx.amount * ctx.scale}px`
    });
    if (!el) return;
    // 四道电弧由 CSS 摆角度，JS 只负责补节点。
    for (let i = 0; i < 4; i++) el.appendChild(document.createElement('i'));
  });

  function group(map, cls) {
    let g = map.querySelector(`.${cls}`);
    if (!g) {
      g = document.createElementNS(NS, 'g');
      g.setAttribute('class', cls);
    }
    return g;
  }

  // 区域轮廓：把节点按极角绕成一圈再连起来，沿半径外扩一点。节点稀疏的
  // 方向补一个向内的凹点，轮廓就凹进去贴住节点群。顶点顺序由调用方按节点
  // 集合缓存下来，这里只负责算坐标，位置抖一下不会重排整圈。
  // 两端圆头的胶囊：两点连线的包围范围。
  function capsule(a, b, width) {
    const r = width / 2;
    let dx = b.x - a.x, dy = b.y - a.y;
    const len = Math.hypot(dx, dy) || 1;
    const nx = -dy / len, ny = dx / len;
    const p = (pt, s) => `${(pt.x + nx * r * s).toFixed(1)} ${(pt.y + ny * r * s).toFixed(1)}`;
    return `M ${p(a, 1)} L ${p(b, 1)} A ${r.toFixed(1)} ${r.toFixed(1)} 0 0 0 ${p(b, -1)} L ${p(a, -1)} A ${r.toFixed(1)} ${r.toFixed(1)} 0 0 0 ${p(a, 1)} Z`;
  }

  function outline(seq, mx, my, grow) {
    const ring = seq.map((p) => ({
      p,
      a: Math.atan2(p.y - my, p.x - mx),
      r: Math.hypot(p.x - mx, p.y - my),
    }));
    let far = 0;
    for (const it of ring) far = Math.max(far, it.r);
    const out = [];
    for (let i = 0; i < ring.length; i++) {
      const cur = ring[i], next = ring[(i + 1) % ring.length];
      let gap = next.a - cur.a;
      if (gap < 0) gap += Math.PI * 2;
      out.push([cur.p.x + Math.cos(cur.a) * grow, cur.p.y + Math.sin(cur.a) * grow]);
      if (gap > Math.PI * 0.7 && gap < Math.PI * 1.6) {
        const mid = cur.a + gap / 2;
        const dent = Math.max(grow * 0.9, Math.min(cur.r, next.r) * 0.72, far * 0.35);
        out.push([mx + Math.cos(mid) * dent, my + Math.sin(mid) * dent]);
      }
    }
    return out;
  }

  // Sutherland–Hodgman：用半平面 nx*x + ny*y + c >= 0 切一刀，保留正侧。
  function clipHalf(pts, nx, ny, c) {
    const out = [];
    for (let i = 0; i < pts.length; i++) {
      const a = pts[i], b = pts[(i + 1) % pts.length];
      const da = nx * a[0] + ny * a[1] + c;
      const db = nx * b[0] + ny * b[1] + c;
      if (da >= 0) out.push(a);
      if ((da >= 0) !== (db >= 0)) {
        const t = da / (da - db);
        out.push([a[0] + (b[0] - a[0]) * t, a[1] + (b[1] - a[1]) * t]);
      }
    }
    return out;
  }

  // 把地块压回后端分给这个区的那块地界：骨干圆内 / 圆外，再夹进两条射线。
  // 骨干圆用 24 边形近似，所以顶点是纯半平面裁剪，不会自交。
  function clipToArea(ring, area, core, rot, slot) {
    let pts = ring;
    const steps = 24;
    if (area === 0) {
      for (let k = 0; k < steps && pts.length >= 3; k++) {
        const a = (k * Math.PI * 2) / steps;
        pts = clipHalf(pts, -Math.cos(a), -Math.sin(a), core);
      }
      return pts;
    }
    if (slot != null) {
      const a0 = rot + (slot * Math.PI * 2) / 3;
      const a1 = a0 + (Math.PI * 2) / 3;
      pts = clipHalf(pts, Math.cos(a0 + Math.PI / 2), Math.sin(a0 + Math.PI / 2), 0);
      pts = clipHalf(pts, Math.cos(a1 - Math.PI / 2), Math.sin(a1 - Math.PI / 2), 0);
    }
    // 圆外不是凸集，半平面裁不出来，改成把落在圆内的顶点沿半径推出去。
    // 先裁扇区再推，扇区内半径是单调的，推完不会自交。
    return pts.map(([x, y]) => {
      const r = Math.hypot(x, y);
      if (r >= core) return [x, y];
      const k = r > 1e-6 ? core / r : 0;
      return [x * k, y * k];
    });
  }

  // 线段是否完全在骨干圆外，与后端 segmentClears 同一判据。
  function clears(a, b, r) {
    const dx = b.x - a.x, dy = b.y - a.y;
    const len2 = dx * dx + dy * dy;
    if (!len2) return Math.hypot(a.x, a.y) > r;
    let t = -(a.x * dx + a.y * dy) / len2;
    t = t < 0 ? 0 : t > 1 ? 1 : t;
    return Math.hypot(a.x + dx * t, a.y + dy * t) > r;
  }

  // 直线会切进骨干圆时改走一个折点，折点贴着圆的肩部，绕行最短。
  // 骨干内部的线直接穿过 —— 那个圆本来就是 area0 自己的地盘。
  // 与后端 network.route 同一套判定，画出来的线就是报文飞的路。
  function bend(a, b, coreR, fieldR) {
    if (!coreR) return null;
    const fa = Math.hypot(a.x, a.y), ta = Math.hypot(b.x, b.y);
    if (fa < coreR * 1.02 && ta < coreR * 1.02) return null;
    if (clears(a, b, coreR * 1.04)) return null;
    const dx = b.x - a.x, dy = b.y - a.y;
    const len2 = dx * dx + dy * dy;
    if (!len2) return null;
    let t = -(a.x * dx + a.y * dy) / len2;
    t = t < 0 ? 0 : t > 1 ? 1 : t;
    const mx = a.x + dx * t, my = a.y + dy * t;
    const near = Math.hypot(mx, my);
    const len = Math.sqrt(len2);
    const nx = near > 1e-6 ? mx / near : -dy / len;
    const ny = near > 1e-6 ? my / near : dx / len;
    for (let push = coreR * 1.12 - near; push <= coreR * 3.5; push += coreR * 0.25) {
      const q = { x: mx + nx * push, y: my + ny * push };
      if (fieldR > 0 && Math.hypot(q.x, q.y) > fieldR * 0.94) break;
      if (clears(a, q, coreR * 1.02) && clears(q, b, coreR * 1.02)) return [q, b];
    }
    return null;
  }

  // 机房布线：每区先铺一层淡色地板把同区节点圈在一起，再用区色把节点拉线
  // 到区内 DR（等于挂上广播网），ABR 另外拉一条琥珀上行干线到最近的骨干节点。
  // 只画不判定，不参与玩法。
  function topology(u, ctx) {
    sync(ctx.time);
    const nodes = (ctx.units || []).filter(
      (n) => n.kind === 'OSPF节点' && n.ownerId === u.id && Number.isFinite(n.x) && Number.isFinite(n.y),
    );
    if (!nodes.length) return;
    const scale = ctx.scale || 1;
    const root = ctx.ensureGuide(`guide-ospf-${u.id}`, 'ospf-topology');
    ctx.seenGuides.add(root.id);
    let map = root.querySelector('svg');
    if (!map) {
      map = document.createElementNS(NS, 'svg');
      map.setAttribute('class', 'ospf-map');
      map.setAttribute('aria-hidden', 'true');
      root.appendChild(map);
    }
    const padsG = group(map, 'ospf-pads');
    const linesG = group(map, 'ospf-lines');
    const tagsG = group(map, 'ospf-tags');
    map.appendChild(padsG);
    map.appendChild(linesG);
    map.appendChild(tagsG);

    const sx = (n) => ctx.cx + n.x * scale;
    const sy = (n) => ctx.cy - n.y * scale;
    const areaOf = new Map(), groups = new Map();
    for (const n of nodes) {
      // 角色事件还没到的节点先不进任何区。以前这里兜底成 area 2，新节点
      // 出生那一两帧会被算进 STUB 的地块里，闪出一个大块下一帧又消失。
      const role = roles.get(n.id);
      if (!role) continue;
      const area = role[0];
      areaOf.set(n.id, area);
      if (!groups.has(area)) groups.set(area, []);
      groups.get(area).push(n);
    }
    const seen = new Set();
    const parts = map._parts || (map._parts = new Map());
    const orders = map._orders || (map._orders = new Map());
    // 分区几何：骨干半径、扇区旋转、扇区→area 的排列。
    const cut = cuts.get(u.id);
    const fieldR = cut ? cut[0] : 0;
    const coreR = fieldR / 2;
    // 区域地盘：同区节点拉一块淡色多边形，只着色不描边，几个区叠起来只是
    // 两片颜色相加。剩一两个点时退成胶囊、圆点。
    const grow = 17 * scale;
    // 后端的扇区排列：code 的三位十进制分别是 slot0→area、slot1→area、slot2→area。
    const rot = cut ? cut[1] : 0;
    const slotOf = new Map();
    if (cut) {
      const code = Math.round(cut[2]) || 0;
      const perm = [code % 10, Math.floor(code / 10) % 10, Math.floor(code / 100) % 10];
      // 后端 at() 返回 1+areas[slot]，所以 area 编号比 perm 里的值大 1。
      perm.forEach((a, sl) => slotOf.set(a + 1, sl));
    }
    // 只画上一帧就已经在场的节点：新节点出生的那一两帧坐标可能还没就位，
    // 让它晚一帧进地块，就不会闪出一下畸形多边形。
    const known = map._known || (map._known = new Set());
    for (const [area, all] of groups) {
      const list = all.filter((n) => known.has(n.id));
      const key = `area:${area}`;
      seen.add(key);
      if (list.length === 0) continue;
      const pts = list.map((n) => ({ x: sx(n), y: sy(n) }));
      const mx = pts.reduce((s, p) => s + p.x, 0) / pts.length;
      const my = pts.reduce((s, p) => s + p.y, 0) / pts.length;
      const kind = pts.length >= 3 ? 'polygon' : (pts.length === 2 ? 'path' : 'circle');
      let el = parts.get(key);
      if (!el || el.tagName.toLowerCase() !== kind) {
        el?.remove();
        el = document.createElementNS(NS, kind);
        parts.set(key, el);
        padsG.appendChild(el);
        orders.delete(area);
      }
      const color = tones[area] || tones[2];
      if (kind === 'polygon') {
        // 顶点顺序按成员集合缓存：只要节点没增减就沿用上一帧的绕行顺序，
        // 免得两个节点角度接近时排序来回跳，把整块面翻成自交的一大块。
        const orderKey = list.map((n) => n.id).sort().join(',');
        let order = orders.get(area);
        // 成员集合变了就重排一次，但这一帧先不重画：新节点可能还没就位，
        // 沿用上一帧的形状能把那一帧的畸形挡掉。
        const stale = !order || order.key !== orderKey;
        if (stale) {
          const ids = list
            .map((n) => ({ id: n.id, a: Math.atan2(sy(n) - my, sx(n) - mx) }))
            .sort((p, q) => p.a - q.a)
            .map((p) => p.id);
          const ring = order ? order.ring : null;
          order = { key: orderKey, ids, ring };
          orders.set(area, order);
        }
        if (stale && order.ring) {
          // 上一帧的形状够用，跳过这一帧的重算。
        } else {
          const byId = new Map(list.map((n) => [n.id, n]));
          const seq = order.ids
            .map((id) => byId.get(id))
            .filter(Boolean)
            .map((n) => ({ x: sx(n), y: sy(n) }));
          let ring = outline(seq, mx, my, grow);
          // 地界裁剪：世界坐标下先把地块切回本区，再换回屏幕坐标。
          if (cut && coreR > 0) {
            const toWorld = ([x, y]) => [(x - ctx.cx) / scale, (ctx.cy - y) / scale];
            const toScreen = ([x, y]) => [ctx.cx + x * scale, ctx.cy - y * scale];
            const clipped = clipToArea(ring.map(toWorld), area, coreR, rot, slotOf.get(area));
            if (clipped.length >= 3) ring = clipped.map(toScreen);
            else ring = null;
          }
          if (!ring) {
            // 被地界整块切掉了，这一帧保持原样。
          } else {
            // 顶点数一变就插值的话，新顶点会被挤到旧顶点旁边，环自交、填充
            // 规则翻面，看起来就是一帧畸形。这种帧直接换形，用一次淡入带过去。
            if (order.ring && order.ring.length === ring.length) {
              ring = ring.map(([x, y], i) => {
                const [px, py] = order.ring[i];
                return [px + (x - px) * 0.4, py + (y - py) * 0.4];
              });
            } else if (order.ring && el.animate) {
              el.animate([{ opacity: 0.3 }, { opacity: 1 }], {
                duration: 260,
                easing: 'ease-out',
              });
            }
            order.ring = ring;
            el.setAttribute('class', 'ospf-area-pad');
            el.setAttribute('points', ring.map(([x, y]) => `${x.toFixed(1)},${y.toFixed(1)}`).join(' '));
            el.setAttribute('fill', color);
          }
        }
      } else if (kind === 'path') {
        el.setAttribute('class', 'ospf-area-pad');
        el.setAttribute('d', capsule(pts[0], pts[1], grow * 1.7));
        el.setAttribute('fill', color);
      } else {
        el.setAttribute('class', 'ospf-area-pad-dot');
        el.setAttribute('cx', mx.toFixed(1));
        el.setAttribute('cy', my.toFixed(1));
        el.setAttribute('r', grow.toFixed(1));
        el.setAttribute('fill', color);
      }
    }
    // 区域铭牌：把 area 编号和类型直接写出来，开局就能读。
    for (const [area, list] of groups) {
      const key = `tag:${area}`;
      seen.add(key);
      const xs = list.map(sx), ys = list.map(sy);
      const half = map.clientWidth * 0.38;
      const tx = Math.min(ctx.cx + half, Math.max(ctx.cx - half, xs.reduce((s, v) => s + v, 0) / xs.length));
      // 别吊出场地：顶到六边形上沿就往下收。
      const ty = Math.max(ctx.cy - map.clientHeight * 0.4, Math.min(...ys) - grow - 6);
      let el = parts.get(key);
      if (!el || el.tagName.toLowerCase() !== 'text') {
        el?.remove();
        el = document.createElementNS(NS, 'text');
        el.setAttribute('class', 'ospf-area-tag');
        parts.set(key, el);
        tagsG.appendChild(el);
      }
      el.setAttribute('x', tx.toFixed(1));
      el.setAttribute('y', ty.toFixed(1));
      const label = `AREA ${area} ${AREA_NAME[area] || ''}`.trim();
      if (el.textContent !== label) el.textContent = label;
      el.setAttribute('fill', tones[area] || tones[2]);
    }
    if (nodes.length >= 2) {
      // 每区先找 DR 当集线器，还没选出 DR 就退回第一个节点。
      const hubs = new Map();
      for (const [area, list] of groups) {
        const dr = list.find((n) => ((roles.get(n.id) || [2, 0])[1] & 1) !== 0);
        hubs.set(area, (dr || list[0]).id);
      }
      const backbone = groups.get(0) || [];
      // 后端跨区干线收包的是 elect 出来的边界路由器（role 的 4 位），不是
      // "最近的骨干节点"——按后者画会有一条报文走在没画的线上。
      const brs = nodes.filter((n) => ((roles.get(n.id) || [2, 0])[1] & 4) !== 0);
      const nearestBackbone = (n) => {
        const pool = brs.length ? brs : backbone;
        return pool.reduce((best, b) => {
          const d = Math.hypot(b.x - n.x, b.y - n.y);
          return !best || d < best.d ? { at: b, d } : best;
        }, null)?.at;
      };
      const byId = new Map(nodes.map((n) => [n.id, n]));
      const trim = 11 * scale;
      const staticSegs = [];
      const link = (a, b, kind, raw) => {
        if (!a || !b) return;
        // 世界坐标下算折点，再转屏幕，保证与后端报文路径同一形状。
        // raw 段来自后端航线，已经是折好的，不能再折一次。
        const world = raw ? [{ x: a.x, y: a.y }, { x: b.x, y: b.y }]
          : [{ x: a.x, y: a.y }, ...(bend(a, b, coreR, fieldR) || [b])];
        const pts = world.map((p) => [sx(p), sy(p)]);
        const total = pts.reduce((sum, p, i) => (i ? sum + Math.hypot(p[0] - pts[i - 1][0], p[1] - pts[i - 1][1]) : 0), 0);
        if (total <= trim * 2) return;
        // 两端各缩一个节点半径，线不压到机架上。
        const cut = (i, j, t) => {
          const [x0, y0] = pts[i], [x1, y1] = pts[j];
          const len = Math.hypot(x1 - x0, y1 - y0);
          if (len <= t) return;
          pts[i] = [x0 + ((x1 - x0) / len) * t, y0 + ((y1 - y0) / len) * t];
        };
        cut(0, 1, trim);
        cut(pts.length - 1, pts.length - 2, trim);
        const key = `line:${kind}:${a.id}:${b.id}`;
        seen.add(key);
        let el = parts.get(key);
        if (!el || el.tagName.toLowerCase() !== 'polyline') {
          el?.remove();
          el = document.createElementNS(NS, 'polyline');
          el.setAttribute('class', CABLE_CLASS[kind] || CABLE_CLASS.inner);
          parts.set(key, el);
          linesG.appendChild(el);
        }
        if (kind !== 'live') staticSegs.push([a.x, a.y, b.x, b.y]);
        if (kind === 'inner') {
          const tone = tones[areaOf.get(a.id)] || tones[2];
          if (el._tone !== tone) { el._tone = tone; el.style.stroke = tone; }
        }
        // points 没变就别碰 DOM：满场三四十条线时每帧重写会让 SVG 反复重排。
        const str = pts.map(([x, y]) => `${x.toFixed(1)},${y.toFixed(1)}`).join(' ');
        if (el._pts !== str) { el._pts = str; el.setAttribute('points', str); }
      };
      // 后端航线是现成的一段，直接画，不折、不缩、不算长度。
      const liveSeg = (key, x1, y1, x2, y2) => {
        seen.add(key);
        let el = parts.get(key);
        if (!el) {
          el = document.createElementNS(NS, 'polyline');
          el.setAttribute('class', CABLE_CLASS.live);
          parts.set(key, el);
          linesG.appendChild(el);
        }
        const str = `${x1.toFixed(1)},${y1.toFixed(1)} ${x2.toFixed(1)},${y2.toFixed(1)}`;
        if (el._pts !== str) { el._pts = str; el.setAttribute('points', str); }
      };
      for (const n of nodes) {
        const area = areaOf.get(n.id);
        const hub = hubs.get(area);
        if (hub != null && hub !== n.id) link(n, byId.get(hub), 'inner');
        if (area !== 0 && ((roles.get(n.id) || [2, 0])[1] & 2)) {
          const up = nearestBackbone(n);
          if (up) link(n, up, 'uplink');
        }
      }
      // 本体发出的第一包走的是"本体→最近的骨干/普通/NSSA 节点"，这条也要画出来。
      const asbr = nodes.reduce((best, n) => {
        if (![0, 1, 3].includes(areaOf.get(n.id))) return best;
        const d = Math.hypot(n.x - u.x, n.y - u.y);
        return !best || d < best.d ? { at: n, d } : best;
      }, null)?.at;
      if (asbr) link(u, asbr, 'inject'); // 本体到第一跳的线也由航线覆盖，这里只当兜底
      // 后端发来的航线：只留最近 3 秒的，跟着报文一起淡出。
      const live = Date.now();
      if (!map._sweep || live - map._sweep > 500) {
        map._sweep = live;
        for (const [key, c] of cables) if (live - c[4] > 4000) cables.delete(key);
      }
      // 静态线已经画过的连接，动态线不再叠一遍：重合时留静态那条。
      const near = (x, y, ux, uy) => Math.hypot(x - ux, y - uy) < 24;
      const covered = (ax, ay, bx, by) => staticSegs.some(([x0, y0, x1, y1]) =>
        (near(ax, ay, x0, y0) && near(bx, by, x1, y1)) ||
        (near(ax, ay, x1, y1) && near(bx, by, x0, y0)));
      if (cables.size) {
        for (const [key, c] of cables) {
          if (covered(c[0], c[1], c[2], c[3])) continue;
          liveSeg('live:' + key, sx({ x: c[0] }), sy({ y: c[1] }), sx({ x: c[2] }), sy({ y: c[3] }));
        }
      }
    }
    // 记下这一帧在场的节点，下一帧才允许它们进地块。
    for (const n of nodes) known.add(n.id);
    for (const [key, el] of parts) if (!seen.has(key)) { el.remove(); parts.delete(key); }
  }

  hooks.ospf = {
    tick(el, u, ctx) {
      sync(ctx.time);
      el.dataset.ospfId = u.id;
      if (!el.querySelector('.ospf-rack')) el.innerHTML = rackFace();
      const [drop, send] = clocks.get(u.id) || [0, 1];
      const install = cycle(drop, ctx.time, 2);
      const outbound = cycle(send, ctx.time, 1);
      el.style.setProperty('--ospf-install', install.toFixed(3));
      el.style.setProperty('--ospf-send', outbound.toFixed(3));
      // 刚装完一台就点一下电源灯，刚发完一包就点一下金手指。
      el.style.setProperty('--ospf-install-pop', clamp01(1 - install / 0.12).toFixed(3));
      el.style.setProperty('--ospf-send-pop', clamp01(1 - outbound / 0.2).toFixed(3));
    },
    unmount(el) {
      clocks.delete(Number(el.dataset.ospfId));
      ['--ospf-install', '--ospf-send', '--ospf-install-pop', '--ospf-send-pop']
        .forEach((key) => el.style.removeProperty(key));
    },
    guide: topology
  };

  hooks['ospf-node'] = {
    tick(el, u, ctx) {
      sync(ctx.time);
      el.dataset.ospfId = u.id;
      if (!el.querySelector('.ospf-switch')) el.innerHTML = switchFace();
      const [area, role] = roles.get(u.id) || [2, 0];
      el.style.setProperty('--ospf-area', tones[area] || tones[2]);
      el.classList.toggle('ospf-dr', !!(role & 1));
      el.classList.toggle('ospf-abr', !!(role & 2));
    },
    unmount(el) {
      roles.delete(Number(el.dataset.ospfId));
      el.classList.remove('ospf-dr', 'ospf-abr');
      el.style.removeProperty('--ospf-area');
    }
  };

  hooks['ospf-shot'] = {
    tick(el, u, ctx) {
      // 报文照中转站的写法：正文两格 01 交替亮，身后三格 010 拖尾顺着
      // 速度方向铺开，颜色仍是 LSA 类型。
      if (!el.querySelector('.ospf-datagram')) {
        el.innerHTML =
          '<span class="ospf-datagram" aria-hidden="true">' +
          '<span class="ospf-packet-tail"><i>0</i><i>1</i><i>0</i></span>' +
          '<b class="ospf-bits"><span>0</span><span>1</span></b></span>';
      }
      const angle = Math.atan2(-(u.vy || 0), u.vx || 0);
      const phase = (((ctx.time || 0) * 6 + (u.id || 0) * 0.17) % 1 + 1) % 1;
      el.style.setProperty('--ospf-angle', angle + 'rad');
      el.style.setProperty('--ospf-font', Math.max(9, (u.radius || 5) * 2 * (ctx.scale || 1)) + 'px');
      el.style.setProperty('--ospf-bit-0', phase < 0.5 ? '1' : '.45');
      el.style.setProperty('--ospf-bit-1', phase < 0.5 ? '.45' : '1');
      el.style.setProperty('--ospf-shift', (phase * 0.4).toFixed(3) + 'em');
    },
  };
})();
