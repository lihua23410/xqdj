(function () {
  window.lookFX = window.lookFX || {};
  window.lookFX.vortex = {
    tick(el) {
      if (!el || el.querySelector(":scope > .vortex-spin")) return;
      const spin = document.createElement("i");
      spin.className = "vortex-spin";
      el.appendChild(spin);
    },
    unmount(el) {
      el.querySelector(":scope > .vortex-spin")?.remove();
    },
  };
})();
