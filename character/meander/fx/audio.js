// 水晶碎响攻击与低柔气流换环；素材处理已烘焙，实时与导出共用同一播放路径。
(function () {
  const A = window.arena;
  const base = document.currentScript?.dataset.pack || "/ball/迷走";
  const definitions = {
    attack: { mode: A.SOUND_MODE.PREEMPT, priority: 199, volume: 1, gap: 0.08 },
    switch: { mode: A.SOUND_MODE.IMMUNE, priority: 0, volume: 1, gap: 0.12 },
  };
  const sounds = {};
  for (const [name, options] of Object.entries(definitions)) {
    const audio = new Audio(`${base}/fx/${name}.wav`);
    audio.preload = "auto";
    audio.load();
    sounds[name] = { voice: A.voice(audio, options), options, last: -Infinity };
  }

  function play(name, fx, ctx) {
    if (ctx?.visualOnly) return;
    const sound = sounds[name];
    // 即时声音回调的 ctx.time 是 0，使用服务端随 FX 带来的对局时间限频。
    const now = Number(fx.amount) || 0;
    if (now >= sound.last && now - sound.last + 1e-9 < sound.options.gap) return;
    sound.last = now;
    sound.voice.stop();
    sound.voice.reset();
    sound.voice.play();
  }

  A.registerShot("attack", (fx, ctx) => play("attack", fx, ctx));
  A.registerShot("meander-switch", (fx, ctx) => play("switch", fx, ctx));
})();
