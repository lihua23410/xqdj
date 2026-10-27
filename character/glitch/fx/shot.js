(function () {
  const el = document.currentScript;
  const base = (el && el.dataset && el.dataset.pack) || "";
  const iaiSrc = `${base}/fx/iai.mp3`;
  const iai = new Audio(iaiSrc);
  iai.preload = "auto";
  iai.load();
  const iaiVoice = arena.voice(iai, { mode: arena.SOUND_MODE.PREEMPT, priority: 1 });

  function unlock() {
    iai.muted = true;
    iai.play()
      .then(() => {
        iai.pause();
        iai.currentTime = 0;
        iai.muted = false;
      })
      .catch(() => {
        iai.muted = false;
      });
  }
  document.addEventListener("pointerdown", unlock, { once: true });

  function playIai() {
    iaiVoice.stop();
    iaiVoice.reset();
    iai.muted = false;
    iai.currentTime = 0;
    iaiVoice.play();
  }

  arena.registerShot("iai", () => {
    playIai();
  });
})();
