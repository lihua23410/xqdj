(function () {
  window.engineHeat = window.engineHeat || {};

  const attackSrc = (document.currentScript && document.currentScript.dataset.pack) || "";
  const attackVoice = arena.voice(new Audio(attackSrc + "/fx/attack.wav"), {
    mode: arena.SOUND_MODE.PREEMPT,
    priority: 199,
    volume: 0.5,
  });
  attackVoice.audio.preload = "auto";

  const blastVoice = arena.voice(new Audio(attackSrc + "/fx/blast.wav"), {
    mode: arena.SOUND_MODE.PREEMPT,
    priority: 60,
  });
  blastVoice.audio.preload = "auto";

  const gearVoice = arena.voice(new Audio(attackSrc + "/fx/gear.wav"), {
    mode: arena.SOUND_MODE.IMMUNE,
  });
  gearVoice.audio.preload = "auto";

  arena.registerShot("gear", () => {
    gearVoice.stop();
    gearVoice.reset();
    gearVoice.play();
  });

  const cruiseVoice = arena.voice(new Audio(attackSrc + "/fx/cruise.wav"), {
    mode: arena.SOUND_MODE.IMMUNE,
  });
  cruiseVoice.audio.preload = "auto";

  arena.registerShot("cruise", () => {
    cruiseVoice.stop();
    cruiseVoice.reset();
    cruiseVoice.play();
  });

  arena.registerShot("attack", () => {
    attackVoice.stop();
    attackVoice.reset();
    attackVoice.play();
  });

  arena.registerShot("heat", (fx) => {
    if (!fx || fx.unitId == null) return;
    window.engineHeat[fx.unitId] = {
      gear: fx.amount || 1,
      weak: fx.vx || 0,
      frail: (fx.vy || 0) > 0,
    };
  });

  arena.registerShot("blast", (fx, ctx) => {
    blastVoice.stop();
    blastVoice.reset();
    blastVoice.play();

    const r = Math.max(48, (fx.amount || 96) * (ctx.scale || 1) * 2);
    const size = { "--r": `${r}px` };
    arena.spawnFx("fx-engine-blast-core", ctx.x, ctx.y, fx.kind);
    arena.spawnFx("fx-engine-blast-wave", ctx.x, ctx.y, fx.kind, size);
    arena.spawnFx("fx-engine-blast-wave", ctx.x, ctx.y, fx.kind, {
      "--r": `${r}px`,
      "--delay": "0.1s",
    });
    arena.spawnFx("fx-engine-blast-ring", ctx.x, ctx.y, fx.kind, size);
    arena.spawnFx("fx-flash", ctx.x, ctx.y, fx.kind);
    arena.spawnFx("fx-shock", ctx.x, ctx.y, fx.kind);
    arena.burst(ctx.x, ctx.y, fx.kind, 28);
  });
})();
