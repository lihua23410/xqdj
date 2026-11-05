// Pack-local, bounded voices. No per-cell sound, global timer, or permanent loop.
(function () {
  const A = window.arena;
  if (!A?.voice || typeof Audio === "undefined") return;
  const base = document.currentScript?.dataset.pack || "/ball/纵火者";
  const definitions = {
    oil: { files: ["oil-break.wav", "oil-break.wav"], gap: .10, priority: 210 },
    fire: { files: ["fire-break.wav", "fire-break.wav"], gap: .15, priority: 210 },
    crackle: { files: ["crackle-a.wav", "crackle-b.wav"], gap: 1.0, priority: 230 },
  };
  const sounds = {};
  for (const [name, options] of Object.entries(definitions)) {
    sounds[name] = { options, last: -Infinity, next: 0, pool: options.files.map(file => {
      const audio = new Audio(`${base}/fx/${file}`);
      audio.preload = "auto";
      return A.voice(audio, { mode: A.SOUND_MODE.NORMAL, priority: options.priority, volume: 1, loop: false });
    }) };
  }
  const seen = new Set();
  let lastTime = -Infinity;
  function sync(time) {
    if (time < lastTime) {
      seen.clear();
      for (const sound of Object.values(sounds)) {
        sound.last = -Infinity; sound.next = 0;
        for (const voice of sound.pool) voice.stop();
      }
    }
    lastTime = time;
  }
  function play(name, time) {
    const sound = sounds[name];
    if (time - sound.last + 1e-9 < sound.options.gap) return;
    sound.last = time;
    const voice = sound.pool[sound.next++ % sound.pool.length];
    voice.stop(); voice.reset(); voice.play();
  }
  function shatter(name, fx, ctx = {}) {
    if (ctx.visualOnly) return;
    const time = Number.isFinite(ctx.time) ? ctx.time : 0;
    sync(time);
    const key = `${name}:${fx.unitId}`;
    if (seen.has(key)) return;
    seen.add(key);
    if (seen.size > 256) seen.delete(seen.values().next().value);
    play(name, time);
  }
  A.registerShot("arson-oil-break", (fx, ctx) => shatter("oil", fx, ctx));
  A.registerShot("arson-fire-break", (fx, ctx) => shatter("fire", fx, ctx));
  window.arsonSFX = {
    observe(ctx, active) {
      const time = Number.isFinite(ctx.time) ? ctx.time : 0;
      sync(time);
      // One short crackle bed for the entire pack, regardless of cells/targets.
      // It naturally ends if the source disappears; no orphaned looping audio.
      if (active) play("crackle", time);
    },
  };
})();
