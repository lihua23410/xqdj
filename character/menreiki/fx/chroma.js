window.lookFX = window.lookFX || {};
window.menreikiHook = window.menreikiHook || {};

arena.registerShot("hook", (fx) => {
  if (!fx || fx.unitId == null) return;
  const n = Math.max(1, Math.min(3, Math.round(fx.amount || 1)));
  window.menreikiHook[fx.unitId] = n;
});

arena.registerShot("share", (fx) => {
  if (!fx || fx.unitId == null) return;
  window.menreikiShare = window.menreikiShare || {};
  window.menreikiShare[fx.unitId] = performance.now();
});

arena.registerShot("stun", (fx) => {
  if (!fx || fx.unitId == null) return;
  window.menreikiStun = window.menreikiStun || {};
  window.menreikiStun[fx.unitId] = performance.now();
  paintStun();
});

function clearStun(id) {
  document.getElementById("u-" + id)?.querySelector(":scope > .menreiki-stun")?.remove();
}

function paintStun() {
  const now = performance.now();
  const map = window.menreikiStun || {};
  for (const id of Object.keys(map)) {
    if (now - map[id] > 250) {
      clearStun(id);
      delete map[id];
      continue;
    }
    const ball = document.getElementById("u-" + id);
    if (!ball) continue;
    let img = ball.querySelector(":scope > .menreiki-stun");
    if (!img) {
      img = document.createElement("img");
      img.className = "menreiki-stun";
      img.src = "/ball/面灵气/status/stg_0.png";
      img.alt = "眩晕";
      ball.appendChild(img);
    }
  }
}

window.lookFX.chroma = {
  unmount(el) {
    el?.querySelector(":scope > .menreiki-hook")?.remove();
    el?.classList.remove("menreiki-share");
    for (const id of Object.keys(window.menreikiStun || {})) clearStun(id);
    window.menreikiStun = {};
  },
  tick(el, u) {
    paintStun();
    if (!el || !el.classList.contains("look-chroma")) return;
    if (!u || u.kind !== "面灵气" || u.role !== "fighter") {
      el.querySelector(":scope > .menreiki-hook")?.remove();
      el.classList.remove("menreiki-share");
      return;
    }
    const sharing = window.menreikiShare && performance.now() - (window.menreikiShare[u.id] || 0) < 200;
    el.classList.toggle("menreiki-share", !!sharing);
    let img = el.querySelector(":scope > .menreiki-hook");
    if (!img) {
      img = document.createElement("img");
      img.className = "menreiki-hook";
      img.alt = "勾";
      el.appendChild(img);
    }
    const n = window.menreikiHook[u.id] || 1;
    const src = `/ball/面灵气/status/hook${n}.png`;
    if (img.dataset.n !== String(n)) {
      img.src = src;
      img.dataset.n = String(n);
    }
  },
};
