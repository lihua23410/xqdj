window.lookFX = window.lookFX || {};
window.wolfRaging = window.wolfRaging || {};

window.lookFX.wolf = {
  unmount(el) {
    el?.classList.remove("wolf-rage");
  },
  tick(el, u) {
    if (!el || !el.classList.contains("look-wolf")) return;
    el.classList.toggle("wolf-rage", !!window.wolfRaging[u.id]);
  },
};
