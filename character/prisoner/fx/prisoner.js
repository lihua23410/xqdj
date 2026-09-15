window.lookFX = window.lookFX || {};
window.prisonerChain = window.prisonerChain || {};
window.prisonerDrops = window.prisonerDrops || {};

const CHAIN_LINKS = 12;

function prisonerPackBase(u) {
  const look = (window.arena && typeof window.arena.lookOf === "function" && window.arena.lookOf(u.kind)) || {};
  return look.base || "/ball/囚徒";
}

function ensureArt(el, cls, src) {
  let img = el.querySelector(`:scope > .${cls}`);
  if (!img) {
    img = document.createElement("img");
    img.className = cls;
    img.alt = "";
    img.src = src;
    el.appendChild(img);
  }
  return img;
}

window.lookFX.prisoner = {
  tick(el, u) {
    if (!el || !el.classList.contains("look-prisoner")) return;
    const marks = (u && u.marks) || [];
    const shocked = marks.some((m) => m && m.kind === "触电" && m.stacks > 0);
    el.classList.toggle("is-shock", shocked);
  },
  guide(u, ctx) {
    if (!u || !ctx || !ctx.ensureGuide || !window.arena) return;
    const st = window.prisonerChain[u.id];
    if (st && Number.isFinite(st.hx) && Number.isFinite(st.hy)) {
      const [x1, y1] = arena.screenPos(u.x, u.y, ctx.scale, ctx.cx, ctx.cy);
      const [x2, y2] = arena.screenPos(st.hx, st.hy, ctx.scale, ctx.cx, ctx.cy);
      const g = ctx.ensureGuide(`guide-chain-${u.id}`, "prisoner-chain");
      ctx.placeSeg(g, x1, y1, x2, y2, u.kind);
      g.style.setProperty("--kind", "#b56a32");
      const reach = Math.max(1e-6, Math.hypot(st.hx - u.x, st.hy - u.y));
      const taut = reach >= 250 - 1e-6;
      g.classList.toggle("is-slack", !taut);
      while (g.children.length < CHAIN_LINKS) {
        g.appendChild(document.createElement("i"));
      }
      while (g.children.length > CHAIN_LINKS) g.lastChild.remove();
      const n = CHAIN_LINKS;
      for (let i = 0; i < n; i++) {
        const el = g.children[i];
        const t = n === 1 ? 0 : i / (n - 1);
        el.style.left = taut ? `${t * 100}%` : `${(0.5 + (t - 0.5) * (reach / 250)) * 100}%`;
        el.className = i === n - 1 ? "hook" : "";
      }
      if (ctx.seenGuides) ctx.seenGuides.add(g.id);
    }
    const now = performance.now();
    for (const [key, d] of Object.entries(window.prisonerDrops)) {
      if (!d || now > d.until) {
        delete window.prisonerDrops[key];
        continue;
      }
      const [x, y] = arena.screenPos(d.x, d.y, ctx.scale, ctx.cx, ctx.cy);
      const g = ctx.ensureGuide(`guide-drop-${key}`, "prisoner-drop");
      g.style.left = `${x}px`;
      g.style.top = `${y}px`;
      g.style.setProperty("--kind", arena.kindColor(d.kind));
      g.dataset.kind = d.kind || "";
      if (ctx.seenGuides) ctx.seenGuides.add(g.id);
    }
  },
};

window.lookFX.cage = {
  unmount(el) {
    el?.querySelector(":scope > .cage-grid")?.remove();
  },
  tick(el) {
    if (!el || !el.classList.contains("look-cage")) return;
    if (el.querySelector(":scope > .cage-grid")) return;
    const grid = document.createElement("span");
    grid.className = "cage-grid";
    grid.innerHTML = `<svg viewBox="0 0 120 120" aria-hidden="true">
      <circle cx="60" cy="60" r="58" fill="none" stroke="#3a3a3a" stroke-width="2"/>
      <circle cx="60" cy="60" r="38" fill="none" stroke="#3a3a3a" stroke-width="1.4"/>
      <circle cx="60" cy="60" r="18" fill="none" stroke="#3a3a3a" stroke-width="1.2"/>
      <g stroke="#3a3a3a" stroke-width="1.4">
        <line x1="60" y1="2" x2="60" y2="118"/>
        <line x1="2" y1="60" x2="118" y2="60"/>
        <line x1="18" y1="18" x2="102" y2="102"/>
        <line x1="102" y1="18" x2="18" y2="102"/>
      </g>
      <g fill="none" stroke="#2a2a2a" stroke-width="1">
        <rect x="8" y="8" width="14" height="14"/>
        <rect x="98" y="8" width="14" height="14"/>
        <rect x="8" y="98" width="14" height="14"/>
        <rect x="98" y="98" width="14" height="14"/>
        <rect x="46" y="46" width="28" height="28"/>
      </g>
    </svg>`;
    el.appendChild(grid);
  },
};

window.lookFX.gallows = {
  unmount(el) {
    el?.querySelector(":scope > .gallows-art")?.remove();
  },
  tick(el, u) {
    if (!el || !el.classList.contains("look-gallows")) return;
    ensureArt(el, "gallows-art", `${prisonerPackBase(u)}/fx/gallows.png`);
  },
};

window.lookFX.chair = {
  unmount(el) {
    el?.querySelector(":scope > .chair-art")?.remove();
  },
  tick(el, u) {
    if (!el || !el.classList.contains("look-chair")) return;
    ensureArt(el, "chair-art", `${prisonerPackBase(u)}/fx/chair.svg`);
  },
};
