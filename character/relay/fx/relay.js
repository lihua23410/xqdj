// IGP router face, port activity and private link-state map. The game still
// chooses its own next hop; only relay-forward may light an active route.
(function () {
  const hooks = window.lookFX = window.lookFX || {};
  const NS = "http://www.w3.org/2000/svg";
  const activity = new Map();
  const graphs = new Map();
  let lastTime = -Infinity;

  function clock(time) {
    const now = Number.isFinite(time) ? time : 0;
    if (now < lastTime) { activity.clear(); graphs.clear(); }
    lastTime = now;
    return now;
  }
  function state(id) {
    if (!activity.has(id)) activity.set(id, { join: -Infinity, tx: -Infinity, rx: -Infinity, target: 0 });
    return activity.get(id);
  }
  function notify(name, fx, time) {
    const now = clock(time);
    const s = state(fx.unitId);
    if (name === "join") s.join = now;
    if (name === "forward") { s.tx = now; s.target = Number(fx.amount) || 0; }
    if (name === "arrive") s.rx = now;
  }

  function face(node) {
    // Ports follow the spherical shell rather than forming a separate chassis.
    const angles = node ? [-27, 0, 27] : [-42, -21, 0, 21, 42];
    const ports = angles.map(a => `<g transform="rotate(${a} 50 50)">
      <path class="relay-switch-ports" d="M45 73h10v10H45z"/>
      <path class="relay-contacts" d="M48 80h4"/>
    </g>`).join('');
    return `<svg class="relay-face" viewBox="0 0 100 100" aria-hidden="true">
      <circle class="relay-bezel" cx="50" cy="50" r="47"/>
      <path class="relay-shell-seam" d="M14 61Q50 78 86 61"/>
      ${ports}
      <path class="relay-vents" d="m19 37 5 2m-7 5 6 2m54-7 5-2m-5 9 6-2"/>
      <path class="relay-uplinks" d="M38 16h9v12h-9zM53 16h9v12h-9z"/>
      <path class="relay-routing" d="M31 44h36l-7-7m7 18H31l7 7"/>
      <circle class="relay-led-link" cx="31" cy="25" r="2.8"/>
      <circle class="relay-led-diag" cx="69" cy="25" r="2.8"/>
    </svg><span class="relay-status" aria-hidden="true"></span>`;
  }
  function router(el, u, ctx, node) {
    const now = clock(ctx.time);
    if (!el.querySelector('.relay-face')) el.innerHTML = face(node);
    const s = activity.get(u.id);
    const tx = Math.max(0, 1 - (now - (s?.tx ?? -Infinity)) / .38);
    const rx = Math.max(0, 1 - (now - (s?.rx ?? -Infinity)) / .5);
    el.style.setProperty('--relay-tx', tx.toFixed(3));
    el.style.setProperty('--relay-rx', rx.toFixed(3));
    el.classList.toggle('relay-online', !!s && now - s.join < .9);
    el.classList.toggle('relay-receiving', rx > .08);
    el.classList.toggle('relay-sending', tx > .08);
  }
  function unmount(el) {
    el.querySelectorAll('.relay-face,.relay-status,.relay-datagram').forEach(n => n.remove());
    el.classList.remove('relay-online', 'relay-receiving', 'relay-sending');
    ['--relay-tx','--relay-rx','--relay-angle','--relay-font','--relay-bit-0','--relay-bit-1','--relay-shift'].forEach(k => el.style.removeProperty(k));
  }
  function svg(tag, attrs) {
    const el = document.createElementNS(NS, tag);
    for (const [key, value] of Object.entries(attrs)) el.setAttribute(key, value);
    return el;
  }
  function topology(u, ctx) {
    const now = clock(ctx.time);
    const units = ctx.units || [];
    const routers = units.filter(n => n.id === u.id || (n.kind === '中转站随从' && n.ownerId === u.id)).sort((a,b) => a.id-b.id);
    const ids = new Set(units.map(n => n.id));
    for (const id of activity.keys()) if (!ids.has(id)) activity.delete(id);
    for (const id of graphs.keys()) if (!ids.has(id)) graphs.delete(id);
    if (routers.length < 2) return;
    const byID = new Map(routers.map(n => [n.id,n]));
    const signature = routers.map(n => n.id).join(',');
    let graph = graphs.get(u.id);
    if (!graph || graph.signature !== signature) {
      const visited = [u];
      const edges = [];
      for (const node of routers) {
        if (node.id === u.id) continue;
        let near = visited[0], d = Infinity;
        for (const p of visited) {
          const dist = Math.hypot(node.x-p.x,node.y-p.y);
          if (dist < d) { near=p; d=dist; }
        }
        edges.push([node.id,near.id]);
        visited.push(node);
      }
      graph = {signature,edges};
      graphs.set(u.id, graph);
    }
    const root = ctx.ensureGuide(`guide-relay-${u.id}`, 'relay-topology');
    ctx.seenGuides.add(root.id);
    let map = root.querySelector('svg');
    if (!map) {
      map = svg('svg', {class:'relay-map', 'aria-hidden':'true'});
      root.appendChild(map);
    }
    const x = n => ctx.cx + n.x*ctx.scale;
    const y = n => ctx.cy - n.y*ctx.scale;
    const seen = new Set();
    const links = map._relayLinks || (map._relayLinks = new Map());
    function line(key, attrs) {
      seen.add(key);
      let el = links.get(key);
      if (!el) { el = svg('line',{}); links.set(key,el); map.appendChild(el); }
      for (const [name,value] of Object.entries(attrs)) el.setAttribute(name,value);
    }
    for (const [aid,bid] of graph.edges) {
      const a = byID.get(aid), b = byID.get(bid);
      line(`neighbor-${aid}`, {x1:x(a), y1:y(a), x2:x(b), y2:y(b), class:'relay-adjacency'});
    }
    for (const a of routers) {
      const s = activity.get(a.id), b = s && byID.get(s.target);
      if (!b || now-s.tx > .7) continue;
      const alpha = Math.max(0,1-(now-s.tx)/.7);
      line(`route-${a.id}`, {x1:x(a),y1:y(a),x2:x(b),y2:y(b),class:'relay-route',opacity:alpha,'stroke-dashoffset':-now*45});
    }
    for (const [key,el] of links) if (!seen.has(key)) { el.remove(); links.delete(key); }
  }
  hooks.relay = { tick: (el,u,ctx) => router(el,u,ctx,false), guide: topology, unmount, notify };
  hooks['relay-node'] = { tick: (el,u,ctx) => router(el,u,ctx,true), unmount };
  hooks['relay-shot'] = {
    tick(el,u,ctx) {
      if (!el.querySelector('.relay-datagram')) el.innerHTML = '<span class="relay-datagram" aria-hidden="true"><span class="relay-packet-tail"><i>0</i><i>1</i><i>0</i></span><b class="relay-bits"><span>0</span><span>1</span></b></span>';
      const angle = Math.atan2(-(u.vy || 0), u.vx || 0);
      const phase = (((ctx.time || 0) * 6 + (u.id || 0) * .17) % 1 + 1) % 1;
      el.style.setProperty('--relay-angle', `${angle}rad`);
      el.style.setProperty('--relay-font', `${Math.max(9, (u.radius || 5) * 2 * (ctx.scale || 1))}px`);
      el.style.setProperty('--relay-bit-0', phase < .5 ? '1' : '.45');
      el.style.setProperty('--relay-bit-1', phase < .5 ? '.45' : '1');
      el.style.setProperty('--relay-shift', `${(phase * .4).toFixed(3)}em`);
    }, unmount
  };
})();
