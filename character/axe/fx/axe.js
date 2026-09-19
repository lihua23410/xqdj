window.lookFX = window.lookFX || {};
window.axeSt = window.axeSt || {};

function axeEnsure(el, cls, html) {
  let n = el.querySelector(`:scope > .${cls}`);
  if (!n) {
    n = document.createElement("div");
    n.className = cls;
    if (html) n.innerHTML = html;
    el.appendChild(n);
  }
  return n;
}

function axeFacingDeg(u, st) {
  const vx = Number(u && u.vx) || 0;
  const vy = Number(u && u.vy) || 0;
  if (Math.hypot(vx, vy) < 1e-6) {
    if (st && Number.isFinite(st.face)) return st.face;
    return Number.isFinite(u && u._axeFace) ? u._axeFace : 0;
  }
  const deg = Math.atan2(vx, vy) * (180 / Math.PI);
  if (st) st.face = deg;
  if (u) u._axeFace = deg;
  return deg;
}

function axePoseCss(pose) {
  return 90 - Number(pose);
}

function axeSrc(st, base) {
  const form = st.form | 0;
  if (form === 2) return `${base}/fx/axe-active.png`;
  if (form === 1) return st.redSword ? `${base}/fx/sword-active.png` : `${base}/fx/sword.png`;
  return st.redShield ? `${base}/fx/sheild-active.png` : `${base}/fx/sheild.png`;
}

function axePhialSrc(st, base) {
  const n = st.phial | 0;
  if (n === 1) return `${base}/fx/three-phial.png`;
  if (n === 2) return `${base}/fx/overcharged-phial.png`;
  if (n === 10) return `${base}/fx/full-phial.png`;
  if (n === 20) return `${base}/fx/enhanced-phial.png`;
  return `${base}/fx/empty-phial.png`;
}

window.lookFX.axe = {
  unmount(el) {
    el?.querySelector(":scope > .axe-art")?.remove();
    el?.querySelector(":scope > .axe-phials")?.remove();
  },
  tick(el, u) {
    if (!el || !el.classList.contains("look-axe")) return;
    const base = (window.arena && window.arena.lookOf && window.arena.lookOf(u.kind).base) || "/ball/盾斧";
    const st = window.axeSt[u.id] || { form: 0, pose: 90, phial: 0 };
    const art = axeEnsure(el, "axe-art");
    let img = art.querySelector(":scope > .axe-weapon");
    if (!img) {
      img = document.createElement("img");
      img.className = "axe-weapon";
      img.alt = "";
      art.appendChild(img);
    }
    const src = axeSrc(st, base);
    if (img.getAttribute("src") !== src) img.src = src;
    img.classList.toggle("is-axe", (st.form | 0) === 2);
    art.style.transform = `rotate(${axeFacingDeg(u, st)}deg)`;
    if (!img._axeTween && typeof gsap !== "undefined") {
      const pose = Number.isFinite(st.pose) ? st.pose : (st.form === 0 ? 90 : 120);
      gsap.set(img, { transformOrigin: "50% 78%", rotation: axePoseCss(pose), scaleY: st.flip || 1 });
    }

    const row = axeEnsure(el, "axe-phials");
    while (row.children.length < 5) {
      const p = document.createElement("img");
      p.alt = "";
      row.appendChild(p);
    }
    const ps = axePhialSrc(st, base);
    for (const p of row.children) {
      if (p.getAttribute("src") !== ps) p.src = ps;
    }
  },
};
