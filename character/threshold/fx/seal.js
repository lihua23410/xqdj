window.lookFX = window.lookFX || {};
window.sealLv = window.sealLv || {};

function sealSides(n) {
  const v = Math.floor(Number(n));
  return v >= 3 ? v : 3;
}

function sealPoints(n, scale, spin) {
  const sides = sealSides(n);
  const k = Number.isFinite(scale) ? scale : 1;
  const turn = Number.isFinite(spin) ? spin : 0;
  const pts = [];
  for (let i = 0; i < sides; i++) {
    const a = turn - Math.PI / 2 + (i * 2 * Math.PI) / sides;
    pts.push(`${(Math.cos(a) * k).toFixed(4)},${(Math.sin(a) * k).toFixed(4)}`);
  }
  return pts.join(" ");
}

function sealProgress(st, time) {
  if (!st || !(st.until > st.born)) return 0;
  const p = (Number(time) - st.born) / (st.until - st.born);
  if (p < 0) return 0;
  if (p > 1) return 1;
  return p;
}

window.lookFX.seal = {
  unmount(el) {
    el.querySelector(":scope > .seal")?.remove();
  },
  tick(el, u, ctx) {
    const id = u && u.id;
    const st = id != null ? window.sealLv[id] : null;
    const sides = st ? st.sides : 3;
    const spin = st ? st.spin : 0;
    const p = sealProgress(st, ctx && ctx.time);
    let svg = el.querySelector(":scope > .seal");
    if (!svg) {
      svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
      svg.setAttribute("class", "seal");
      svg.setAttribute("viewBox", "-1.15 -1.15 2.3 2.3");
      el.appendChild(svg);
    }
    svg.innerHTML =
      `<polygon points="${sealPoints(sides, 1, spin)}" fill="none" stroke="#d6ef6a" stroke-width="0.045"/>` +
      `<polygon points="${sealPoints(sides, p, spin)}" fill="rgba(214,239,106,0.28)" stroke="none"/>`;
  },
};

arena.registerShot("seal", (fx) => {
  if (!fx || fx.unitId == null) return;
  const prev = window.sealLv[fx.unitId] || {};
  window.sealLv[fx.unitId] = {
    sides: sealSides(fx.amount),
    born: Number(fx.vx) || 0,
    until: Number(fx.vy) || 0,
    spin: Number.isFinite(prev.spin) ? prev.spin : 0,
  };
});

arena.registerShot("seal-turn", (fx) => {
  if (!fx || fx.unitId == null) return;
  const prev = window.sealLv[fx.unitId] || {};
  prev.spin = Number(fx.amount) || 0;
  window.sealLv[fx.unitId] = prev;
});

arena.registerShot("seal-pop", (fx, ctx) => {
  const root = document.getElementById("over");
  if (!root || !ctx) return;
  const reach = Number(fx && fx.vx);
  const r = (Number.isFinite(reach) && reach > 0 ? reach : 48) * (ctx.scale || 1);
  const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  svg.setAttribute("class", "seal-pop");
  svg.setAttribute("viewBox", "-1.15 -1.15 2.3 2.3");
  svg.style.left = `${ctx.x - r}px`;
  svg.style.top = `${ctx.y - r}px`;
  svg.style.width = `${r * 2}px`;
  svg.style.height = `${r * 2}px`;
  svg.innerHTML = `<polygon points="${sealPoints(fx && fx.amount, 1, fx && fx.vy)}" fill="rgba(214,239,106,0.55)" stroke="#d6ef6a" stroke-width="0.05"/>`;
  root.appendChild(svg);
  setTimeout(() => svg.remove(), 180);
});
