(function () {
  const el = document.currentScript;
  const base = (el && el.dataset && el.dataset.pack) || "";
  const howlSrc = `${base}/fx/animals.mp3`;
  const cutinSrc = `${base}/fx/wolf.jpg`;
  const howl = new Audio(howlSrc);
  howl.preload = "auto";
  howl.load();

  function unlock() {
    howl.muted = true;
    howl.play()
      .then(() => {
        howl.pause();
        howl.currentTime = 0;
        howl.muted = false;
      })
      .catch(() => {
        howl.muted = false;
      });
  }
  document.addEventListener("pointerdown", unlock, { once: true });

  function playHowl() {
    howl.pause();
    howl.muted = false;
    howl.currentTime = 0;
    const start = howl.play();
    if (start && typeof start.catch === "function") start.catch(() => {});
  }

  function showCutin() {
    document.querySelectorAll(".wolf-cutin").forEach((n) => n.remove());
    const root = document.createElement("div");
    root.className = "wolf-cutin";
    const img = document.createElement("img");
    img.src = cutinSrc;
    img.alt = "";
    img.width = 972;
    img.height = 1145;
    root.appendChild(img);
    document.body.appendChild(root);
    root.addEventListener("animationend", (ev) => {
      if (ev.target === root) root.remove();
    });
  }

  window.wolfMoonPhase = window.wolfMoonPhase || {};
  window.wolfRageUntil = window.wolfRageUntil || {};

  arena.registerShot("phase", (fx, ctx) => {
    window.wolfMoonPhase[fx.slot] = fx.amount | 0;
    arena.spawnFx("fx-ring", ctx.x, ctx.y, fx.kind);
  });

  arena.registerShot("rage", (fx) => {
    window.wolfRageUntil[fx.unitId] = performance.now() + 5000;
    playHowl();
    showCutin();
  });
})();
