// 反重力猫咪：出弹闪光、20 秒重力脉冲、弹体方向扭转，以及猫叫和星光音效。
(function () {
  const A = window.arena;
  const base = document.currentScript?.dataset.pack || "/ball/反重力猫咪";
  const sounds = {};
  const definitions = {
    meow: { mode: A.SOUND_MODE.IMMUNE, volume: 1.0, voices: 2, gap: 0.75 },
    twinkle: { mode: A.SOUND_MODE.NORMAL, priority: 205, volume: 0.5, voices: 3, gap: 0.12 },
    starlight: { mode: A.SOUND_MODE.IMMUNE, volume: 0.6, voices: 2, gap: 0.35 },
  };
  for (const [name, options] of Object.entries(definitions)) {
    const pool = Array.from({ length: options.voices }, () => {
      const audio = new Audio(base + "/fx/" + name + ".wav");
      audio.preload = "auto";
      return A.voice(audio, options);
    });
    sounds[name] = { pool, next: 0, last: -Infinity, gap: options.gap, volume: options.volume };
  }

  function play(name, ctx, gain = 1) {
    const sound = sounds[name];
    const now = Number.isFinite(ctx?.time) ? ctx.time : performance.now() / 1000;
    const elapsed = now - sound.last;
    // One cue for a batch of stars; the same match clock also drives video exports.
    if (elapsed >= 0 && elapsed + 1e-9 < sound.gap) return;
    sound.last = now;
    const voice = sound.pool[sound.next++ % sound.pool.length];
    voice.stop();
    voice.reset();
    voice.audio.volume = sound.volume * gain;
    voice.play();
  }

  A.registerShot("hajimi-meow", (fx, ctx) => play("meow", ctx));

  A.registerShot("hajimi-fire", (fx, ctx) => {
    play("twinkle", ctx);
    A.spawnFx("fx-hajimi-fire", ctx.x, ctx.y, ctx.kind);
  });

  A.registerShot("gravity-field", (fx, ctx) => {
    play("meow", ctx);
    play("starlight", ctx);
    const root = document.getElementById("fx");
    if (!root) return;
    const scale = ctx.scale || 1;
    const diameter = Math.max(80, (Number(fx.amount) || 420) * 2 * scale);
    const duration = Math.max(0.2, Number(fx.vx) || 3);
    A.spawnFx("fx-hajimi-field", ctx.x, ctx.y, ctx.kind, {
      "--r": `${diameter}px`,
      "--duration": `${duration}s`,
    }, root);
    A.spawnFx("fx-hajimi-field-ring", ctx.x, ctx.y, ctx.kind, {
      "--duration": `${duration}s`,
      "--r": `${diameter}px`,
      "--delay": "0.12s",
    }, root);
    A.burst(ctx.x, ctx.y, ctx.kind, 28);
  });

  A.registerShot("gravity-twist", (fx, ctx) => {
    play("twinkle", ctx, 0.5);
    const root = document.getElementById("fx");
    if (!root) return;
    const ang = Math.atan2(-(Number(fx.vy) || 0), Number(fx.vx) || 1);
    A.spawnFx("fx-hajimi-twist", ctx.x, ctx.y, ctx.kind, { "--ang": `${ang}rad` }, root);
  });

  A.registerShot("gravity-convert", (fx, ctx) => {
    play("twinkle", ctx, 0.75);
    const root = document.getElementById("fx");
    if (!root) return;
    const ang = Math.atan2(-(Number(fx.vy) || 0), Number(fx.vx) || 1);
    A.spawnFx("fx-hajimi-twist", ctx.x, ctx.y, ctx.kind, { "--ang": `${ang}rad` }, root);
    A.spawnFx("fx-flash", ctx.x, ctx.y, ctx.kind);
    A.burst(ctx.x, ctx.y, ctx.kind, 10);
  });

  A.registerShot("gravity-capture", (fx, ctx) => {
    play("twinkle", ctx, 0.65);
    const root = document.getElementById("fx");
    if (!root) return;
    const ang = Math.atan2(-(Number(fx.vy) || 0), Number(fx.vx) || 1);
    A.spawnFx("fx-hajimi-twist", ctx.x, ctx.y, ctx.kind, { "--ang": `${ang}rad` }, root);
  });

  A.registerShot("gravity-release", (fx, ctx) => {
    play("starlight", ctx, 0.8);
    const root = document.getElementById("fx");
    if (!root) return;
    const ang = Math.atan2(-(Number(fx.vy) || 0), Number(fx.vx) || 1);
    A.spawnFx("fx-hajimi-twist", ctx.x, ctx.y, ctx.kind, { "--ang": `${ang}rad` }, root);
    A.spawnFx("fx-hajimi-fire", ctx.x, ctx.y, ctx.kind);
    A.burst(ctx.x, ctx.y, ctx.kind, 8);
  });
})();
