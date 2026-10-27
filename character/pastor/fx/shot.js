(function () {
  const el = document.currentScript;
  const base = (el && el.dataset && el.dataset.pack) || "";
  const verseVoices = [0, 1, 2, 3].map((i) => {
    const a = new Audio(base + "/fx/verse" + i + ".mp3");
    a.preload = "auto";
    return arena.voice(a, { mode: arena.SOUND_MODE.IMMUNE, priority: 0 });
  });
  const screamVoice = arena.voice(new Audio(base + "/fx/scream.mp3"), {
    mode: arena.SOUND_MODE.PREEMPT,
    priority: 200,
  });
  screamVoice.audio.preload = "auto";
  if (window.arena && typeof arena.registerOwnHurt === "function") arena.registerOwnHurt("老牧师");

  window.pastorFX = window.pastorFX || {};
  window.pastorFX.screaming = window.pastorFX.screaming || {};

  let screamFor = 0;
  let screamLeft = 0;
  let verseOn = -1;
  let last = 0;

  function stopped() {
    return !!document.querySelector(".hitstop");
  }

  function playVerse(i) {
    for (const voice of verseVoices) {
      voice.stop();
      voice.audio.currentTime = 0;
    }
    verseOn = i;
    const voice = verseVoices[i];
    if (!voice) return;
    voice.reset();
    voice.audio.currentTime = 0;
    voice.play();
  }

  function playScream(id) {
    screamVoice.stop();
    screamVoice.reset();
    screamVoice.audio.currentTime = 0;
    screamFor = id;
    screamLeft = 0.696599;
    window.pastorFX.screaming[id] = true;
    screamVoice.play();
  }

  window.pastorFX.tickAudio = function () {
    const now = performance.now();
    const dt = last ? Math.min(0.05, (now - last) / 1000) : 0;
    last = now;
    const verseVoice = verseOn >= 0 ? verseVoices[verseOn] : null;
    if (stopped()) {
      if (!screamVoice.audio.paused) screamVoice.pause();
      if (verseVoice && !verseVoice.audio.paused) verseVoice.pause();
      return;
    }
    if (verseVoice && verseVoice.audio.paused && verseVoice.audio.currentTime > 0 && !verseVoice.audio.ended) {
      verseVoice.play();
    }
    if (!screamFor) return;
    if (screamVoice.audio.paused && screamLeft > 0) screamVoice.play();
    screamLeft -= dt;
    if (screamLeft <= 0) {
      screamVoice.stop();
      delete window.pastorFX.screaming[screamFor];
      screamFor = 0;
    }
  };

  arena.registerShot("verse", (fx) => {
    playVerse(Math.round(fx.amount || 0));
  });
  arena.registerShot("scream", (fx) => {
    if (!fx) return;
    playScream(fx.unitId);
  });
})();
