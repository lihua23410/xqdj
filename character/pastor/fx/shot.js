(function () {
  const el = document.currentScript;
  const base = (el && el.dataset && el.dataset.pack) || "";
  const verses = [0, 1, 2, 3].map((i) => {
    const a = new Audio(base + "/fx/verse" + i + ".mp3");
    a.preload = "auto";
    return a;
  });
  const scream = new Audio(base + "/fx/scream.mp3");
  scream.preload = "auto";

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
    for (const a of verses) {
      a.pause();
      a.currentTime = 0;
    }
    verseOn = i;
    const a = verses[i];
    if (!a) return;
    a.play().catch(() => {});
  }

  function playScream(id) {
    scream.pause();
    scream.currentTime = 0;
    screamFor = id;
    screamLeft = 0.696599;
    window.pastorFX.screaming[id] = true;
    scream.play().catch(() => {});
  }

  window.pastorFX.tickAudio = function () {
    const now = performance.now();
    const dt = last ? Math.min(0.05, (now - last) / 1000) : 0;
    last = now;
    const verse = verseOn >= 0 ? verses[verseOn] : null;
    if (stopped()) {
      if (!scream.paused) scream.pause();
      if (verse && !verse.paused) verse.pause();
      return;
    }
    if (verse && verse.paused && verse.currentTime > 0 && !verse.ended) {
      verse.play().catch(() => {});
    }
    if (!screamFor) return;
    if (scream.paused && screamLeft > 0) scream.play().catch(() => {});
    screamLeft -= dt;
    if (screamLeft <= 0) {
      scream.pause();
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
