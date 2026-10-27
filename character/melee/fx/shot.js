(function () {
  const src = (document.currentScript && document.currentScript.dataset.pack) || "";
  const dashVoice = arena.voice(new Audio(src + "/fx/dash.wav"), {
    mode: arena.SOUND_MODE.IMMUNE,
    priority: 0,
  });
  dashVoice.audio.preload = "auto";

  const attackVoice = arena.voice(new Audio(src + "/fx/attack.wav"), {
    mode: arena.SOUND_MODE.PREEMPT,
    priority: 199,
  });
  attackVoice.audio.preload = "auto";

  arena.registerShot("attack", () => {
    attackVoice.stop();
    attackVoice.reset();
    attackVoice.play();
  });

  arena.registerShot("dash", (fx, ctx) => {
    dashVoice.stop();
    dashVoice.reset();
    dashVoice.play();

    const { x, y, kind } = ctx;
    arena.spawnFx("fx-shock", x, y, kind);
    arena.spawnFx("fx-ring", x, y, kind);
    arena.burst(x, y, kind, 12);
  });
})();
