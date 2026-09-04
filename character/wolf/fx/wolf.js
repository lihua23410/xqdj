window.lookFX = window.lookFX || {};
window.wolfRageUntil = window.wolfRageUntil || {};

window.lookFX.wolf = {
  unmount(el) {
    el?.classList.remove("wolf-rage");
  },
  tick(el, u, ctx) {
    if (!el || !el.classList.contains("look-wolf")) return;
    const until = window.wolfRageUntil[u.id] || 0;
    const on = (ctx && ctx.now || performance.now()) < until;
    el.classList.toggle("wolf-rage", on);
  },
};
