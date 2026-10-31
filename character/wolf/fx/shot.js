(function () {
  const el = document.currentScript;
  const base = (el && el.dataset && el.dataset.pack) || "";
  const howlSrc = `${base}/fx/animals.mp3`;
  const cutinSrc = `${base}/fx/wolf.jpg`;
  const howl = new Audio(howlSrc);
  howl.preload = "auto";
  howl.preservesPitch = true;
  howl.webkitPreservesPitch = true;
  howl.load();
  const howlVoice = arena.voice(howl, { mode: arena.SOUND_MODE.IMMUNE });

  let howlGen = 0;
  let howling = false;
  let howlPlays = 0;

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
    howlGen++;
    howling = true;
    howlVoice.stop();
    howlVoice.reset();
    howl.muted = false;
    howl.playbackRate = 1 + 0.2 * howlPlays;
    howlPlays++;
    howl.currentTime = 0;
    howlVoice.play();
  }

  function stopHowl() {
    howlGen++;
    howling = false;
    howlVoice.stop();
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

  // 月相反馈由月亮皮肤在真实落点显示，不在场心额外画圈。
  arena.registerShot("phase", () => {});

  window.wolfRaging = window.wolfRaging || {};

  arena.registerShot("howl", (fx, ctx) => {
    if (!arena || !arena.spawnFx) return;
    const scale = ctx.scale || 1;
    const lift = 18 * scale * 1.5 + 14;
    const el = arena.spawnFx("fx-wolf-howl", ctx.x, ctx.y - lift, fx.kind);
    if (!el) return;
    el.style.color = "#fff";
    el.textContent = "嗷唔";
  });

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
