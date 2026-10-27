(function () {
  const src = (document.currentScript && document.currentScript.dataset.pack) || "";
  const shotVoice = arena.voice(new Audio(src + "/fx/shot.wav"), {
    mode: arena.SOUND_MODE.IMMUNE,
    priority: 0,
  });
  shotVoice.audio.preload = "auto";

  arena.registerShot("shot", (fx, ctx) => {
    const audioOnly = !!(ctx && ctx.audioOnly);
    const visualOnly = !!(ctx && ctx.visualOnly);
    if (!visualOnly) {
      shotVoice.stop();
      shotVoice.reset();
      shotVoice.play();
    }
    if (audioOnly) return;

    const { x, y, kind } = ctx;
    arena.spawnFx("fx-flash", x, y, kind);
    const ang = Math.atan2(-(fx.vy || 0), fx.vx || 1) * (180 / Math.PI);
    const beam = arena.spawnFx("fx-beam", x, y, kind);
    if (beam) beam.style.transform = `translate(0, -50%) rotate(${ang}deg)`;
  });
})();
