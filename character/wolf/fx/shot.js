(function () {
  const el = document.currentScript;
  const base = (el && el.dataset && el.dataset.pack) || "";
  const howlSrc = `${base}/fx/animals.mp3`;
  const cutinSrc = `${base}/fx/wolf.jpg`;
  const howl = new Audio(howlSrc);
  howl.preload = "auto";
  howl.load();

  let howlGen = 0;
  let howling = false;

  function unlock() {
    const gen = howlGen;
    howl.muted = true;
    howl.play()
      .then(() => {
        if (gen !== howlGen || howling) {
          howl.muted = false;
          return;
        }
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
    const gen = ++howlGen;
    howling = true;
    howl.pause();
    howl.muted = false;
    howl.currentTime = 0;
    const start = howl.play();
    if (start && typeof start.then === "function") {
      start.then(() => {
        if (gen !== howlGen || !howling) {
          howl.pause();
          howl.currentTime = 0;
        }
      }).catch(() => {});
    }
  }

  function stopHowl() {
    howlGen++;
    howling = false;
    howl.pause();
    howl.currentTime = 0;
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

  arena.registerShot("phase", (fx, ctx) => {
    window.wolfMoonPhase[fx.slot] = fx.amount | 0;
    arena.spawnFx("fx-ring", ctx.x, ctx.y, fx.kind);
  });

  window.wolfRaging = window.wolfRaging || {};

  arena.registerShot("rage", (fx) => {
    if (fx.amount > 0) {
      window.wolfRaging[fx.unitId] = true;
      playHowl();
      showCutin();
      return;
    }
    if (fx.amount !== 0) return;
    window.wolfRaging[fx.unitId] = false;
    const still = Object.keys(window.wolfRaging).some((id) => window.wolfRaging[id]);
    if (!still) stopHowl();
  });
})();
