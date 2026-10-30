// 各短音预先处理为 WAV；共用现有仲裁，不使用循环音或运行时变调。
(function () {
  const A = window.arena;
  const base = document.currentScript?.dataset.pack || "/ball/教父(青年)";
  const definitions = {
    pistol: { mode: A.SOUND_MODE.PREEMPT, priority: 199, volume: 0.65, voices: 2, gap: 0.075 },
    inject: { mode: A.SOUND_MODE.IMMUNE, volume: 0.55, voices: 2, gap: 0.3 },
    rush: { mode: A.SOUND_MODE.NORMAL, priority: 205, volume: 0.45, voices: 2, gap: 0.12 },
  };
  const sounds = {};
  for (const [name, options] of Object.entries(definitions)) {
    sounds[name] = { options, last: -Infinity, next: 0,
      pool: Array.from({ length: options.voices }, () => {
        const audio = new Audio(`${base}/fx/${name}.wav`);
        audio.preload = "auto";
        return A.voice(audio, options);
      }),
    };
  }
  window.youthSFX = {
    play(name, time) {
      const sound = sounds[name];
      if (!sound) return;
      const now = Number.isFinite(time) ? time : 0;
      if (now >= sound.last && now - sound.last + 1e-9 < sound.options.gap) return;
      sound.last = now;
      const voice = sound.pool[sound.next++ % sound.pool.length];
      voice.stop();
      voice.reset();
      voice.play();
    },
    reset() {
      for (const sound of Object.values(sounds)) {
        sound.last = -Infinity;
        sound.next = 0;
        for (const voice of sound.pool) voice.stop();
      }
    },
  };
})();
