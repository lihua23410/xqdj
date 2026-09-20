window.lookFX = window.lookFX || {};

function ensureMiuHP(el) {
  let wrap = el.querySelector(":scope > .miu-hp");
  if (wrap) return wrap;
  wrap = document.createElement("span");
  wrap.className = "miu-hp";
  const i = document.createElement("i");
  wrap.appendChild(i);
  el.appendChild(wrap);
  return wrap;
}

window.lookFX.miu = {
  unmount(el) {
    el?.querySelector(":scope > .miu-hp")?.remove();
  },
  tick(el, u) {
    if (!el || !u) return;
    const wrap = ensureMiuHP(el);
    const i = wrap.querySelector("i");
    const max = u.maxHp || 1;
    const hp = Math.max(0, u.hp || 0);
    if (i) i.style.width = `${Math.max(0, Math.min(100, (hp / max) * 100))}%`;
  },
};
