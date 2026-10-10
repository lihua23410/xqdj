window.lookFX = window.lookFX || {};
window.twinAttract = window.twinAttract || {};

(function () {
  const attractR = 80;
  const idleR = 72;
  const curveCount = 4;

  window.lookFX.pull = {
    guide: ringGuide("pull"),
    tick(el) {
      el?.querySelector(":scope > .field-ring")?.remove();
    },
  };
  window.lookFX.push = {
    guide: ringGuide("push"),
    tick(el) {
      el?.querySelector(":scope > .field-ring")?.remove();
    },
  };

  function ringGuide(mode) {
    return function (u, ctx) {
      if (!u || !ctx || !ctx.ensureGuide || !window.arena) return;
      const id = `guide-twin-ring-${u.id}`;
      const g = ctx.ensureGuide(id, `twin-ring ${mode}`);
      g.className = `guide twin-ring ${mode}`;
      if (!g.querySelector(":scope > i")) {
        g.appendChild(document.createElement("i"));
        g.appendChild(document.createElement("b"));
      }
      const attracting = !!window.twinAttract[u.id];
      g.classList.toggle("attract", attracting);
      const scale = ctx.scale || 1;
      const fr = (attracting ? attractR : idleR) * scale;
      // 必须 important：压过 .guide { height:0 }，且避免再被别处 !important 盖掉
      g.style.setProperty("width", `${fr * 2}px`, "important");
      g.style.setProperty("height", `${fr * 2}px`, "important");
      g.style.left = `${ctx.cx + u.x * scale}px`;
      g.style.top = `${ctx.cy - u.y * scale}px`;
      g.style.removeProperty("--ang");
      const color = window.arena.kindColor(u.kind);
      g.style.setProperty("--kind", color);
      g.style.color = color;
      if (ctx.seenGuides) ctx.seenGuides.add(id);
    };
  }

  window.lookFX.bond = {
    guide(u, ctx) {
      if (!u || !ctx || !ctx.ensureGuide || !ctx.placeSeg || !window.arena) return;
      const fx = (window.arena.lookOf(u.kind).fx || []);
      if (!fx.includes("bond")) return;
      const other = (ctx.units || []).find(
        (o) => o.slot === u.slot && o.id !== u.id && (window.arena.lookOf(o.kind).fx || []).includes("bond")
      );
      if (!other || u.id > other.id) return;
      const attracting = !!window.twinAttract[u.id] || !!window.twinAttract[other.id];
      const [x1, y1] = window.arena.screenPos(u.x, u.y, ctx.scale, ctx.cx, ctx.cy);
      const [x2, y2] = window.arena.screenPos(other.x, other.y, ctx.scale, ctx.cx, ctx.cy);
      if (attracting) {
        placeAttractCurves(ctx, u.slot, x1, y1, x2, y2);
        return;
      }
      const g = ctx.ensureGuide(`guide-bond-${u.slot}`, "bond");
      g.className = "guide bond";
      ctx.placeSeg(g, x1, y1, x2, y2, u.kind);
      g.style.setProperty("--kind", "#b44cff");
      const dist = Math.hypot(u.x - other.x, u.y - other.y);
      g.style.opacity = dist < 56 ? "0.9" : "0.42";
      if (ctx.seenGuides) ctx.seenGuides.add(g.id);
    },
  };

  function placeAttractCurves(ctx, slot, x1, y1, x2, y2) {
    const root = document.getElementById("guides");
    if (!root) return;
    const dx = x2 - x1;
    const dy = y2 - y1;
    const len = Math.hypot(dx, dy) || 1;
    const nx = -dy / len;
    const ny = dx / len;
    const midAmp = Math.min(48, Math.max(16, len * 0.34));
    const t = performance.now() * 0.001;
    for (let i = 0; i < curveCount; i++) {
      const id = `guide-bond-curve-${slot}-${i}`;
      let g = document.getElementById(id);
      if (!g) {
        g = document.createElement("div");
        g.id = id;
        g.innerHTML =
          '<svg xmlns="http://www.w3.org/2000/svg" width="1" height="1" preserveAspectRatio="none"><path/></svg>';
        root.appendChild(g);
      }
      g.className = "guide bond-curve";
      const phase = t * (1.2 + i * 0.21) + i * 1.1;
      const side = i % 2 === 0 ? 1 : -1;
      const bend =
        side * midAmp * (0.55 + 0.45 * Math.sin(phase)) * (0.65 + 0.35 * ((i + 1) / curveCount));
      const c1x = x1 + dx * 0.3 + nx * bend;
      const c1y = y1 + dy * 0.3 + ny * bend;
      const c2x = x1 + dx * 0.7 - nx * bend * 0.9;
      const c2y = y1 + dy * 0.7 - ny * bend * 0.9;
      const pad = 12;
      const minX = Math.min(x1, x2, c1x, c2x) - pad;
      const minY = Math.min(y1, y2, c1y, c2y) - pad;
      const w = Math.max(2, Math.max(x1, x2, c1x, c2x) + pad - minX);
      const h = Math.max(2, Math.max(y1, y2, c1y, c2y) + pad - minY);
      g.style.left = `${minX}px`;
      g.style.top = `${minY}px`;
      g.style.setProperty("width", `${w}px`, "important");
      g.style.setProperty("height", `${h}px`, "important");
      g.style.opacity = String(0.7 + 0.25 * Math.sin(phase * 1.3));
      g.style.removeProperty("--ang");
      const svg = g.querySelector("svg");
      const path = g.querySelector("path");
      if (svg && path) {
        svg.setAttribute("viewBox", `0 0 ${w} ${h}`);
        svg.setAttribute("width", String(w));
        svg.setAttribute("height", String(h));
        svg.style.width = `${w}px`;
        svg.style.height = `${h}px`;
        const p = (x, y) => `${(x - minX).toFixed(2)},${(y - minY).toFixed(2)}`;
        path.setAttribute("d", `M ${p(x1, y1)} C ${p(c1x, c1y)} ${p(c2x, c2y)} ${p(x2, y2)}`);
      }
      if (ctx.seenGuides) ctx.seenGuides.add(id);
    }
  }

  arena.registerShot("twin-attract", (fx) => {
    if (!fx || fx.unitId == null) return;
    if ((fx.amount || 0) > 0) window.twinAttract[fx.unitId] = true;
    else delete window.twinAttract[fx.unitId];
  });
})();
