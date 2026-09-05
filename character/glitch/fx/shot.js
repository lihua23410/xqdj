(function () {
  const el = document.currentScript;
  const base = (el && el.dataset && el.dataset.pack) || "";
  const iaiSrc = `${base}/fx/iai.mp3`;
  const iai = new Audio(iaiSrc);
  iai.preload = "auto";
  iai.load();

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
    iai.pause();
    iai.muted = false;
    iai.currentTime = 0;
    const start = iai.play();
    if (start && typeof start.catch === "function") start.catch(() => {});
  }

  arena.registerShot("iai", () => {
    playIai();
  });
})();
