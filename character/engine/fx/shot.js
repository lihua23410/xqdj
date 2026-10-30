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
    arena.spawnFx("fx-engine-blast-bloom", ctx.x, ctx.y, fx.kind, size);
    arena.spawnFx("fx-engine-blast-wave", ctx.x, ctx.y, fx.kind, size);
    arena.spawnFx("fx-engine-blast-wave", ctx.x, ctx.y, fx.kind, {
      "--r": `${r}px`,
      "--delay": "0.09s",
    });
    arena.spawnFx("fx-engine-blast-ring", ctx.x, ctx.y, fx.kind, size);

    // Four cylinder vents, followed by cooling smoke and thrown metal.
    for (let i = 0; i < 4; i++) {
      arena.spawnFx("fx-engine-blast-plume", ctx.x, ctx.y, fx.kind, {
        "--angle": `${i * 90}deg`,
        "--length": `${r * (i % 2 ? 0.36 : 0.42)}px`,
        "--width": `${r * 0.13}px`,
        "--delay": `${i % 2 * 0.025}s`,
      });
    }
    for (let i = 0; i < 8; i++) {
      const a = i * Math.PI / 4 + 0.2;
      const distance = r * (0.19 + i % 3 * 0.035);
      arena.spawnFx("fx-engine-blast-smoke", ctx.x, ctx.y, fx.kind, {
        "--dx": `${Math.cos(a) * distance}px`,
        "--dy": `${Math.sin(a) * distance}px`,
        "--size": `${r * (0.13 + i % 3 * 0.025)}px`,
        "--delay": `${0.08 + i % 3 * 0.025}s`,
      });
    }
    for (let i = 0; i < 26; i++) {
      const shard = i % 3 === 0;
      const a = i * 2.39996 + (fx.unitId || 0) * 0.37;
      const distance = r * (0.23 + (i * 7 % 11) * 0.025);
      arena.spawnFx(shard ? "fx-engine-blast-shard" : "fx-engine-blast-spark", ctx.x, ctx.y, fx.kind, {
        "--dx": `${Math.cos(a) * distance}px`,
        "--dy": `${Math.sin(a) * distance}px`,
        "--angle": `${a * 180 / Math.PI}deg`,
        "--turn": `${(i % 2 ? -1 : 1) * (160 + i * 21)}deg`,
        "--size": `${(ctx.scale || 1) * (shard ? 6 + i % 4 * 1.2 : 2)}px`,
        "--delay": `${i % 4 * 0.012}s`,
      });
    }
  });
})();
