(function () {
  const car = `<svg viewBox="0 0 64 34" aria-hidden="true">
    <path fill="#101827" d="M11 1h10v6H11zM44 1h10v6H44zM11 27h10v6H11zM44 27h10v6H44z"/>
    <path fill="#243755" stroke="#9baabd" stroke-width="1.3" d="M7 5h45l8 5v14l-8 5H7l-4-4V9z"/>
    <path fill="#e7edf1" d="M8 7h43v20H8z"/>
    <path fill="#29456b" d="M11 7h40v5H11zm0 15h40v5H11z"/>
    <path fill="#344c66" d="M17 9h22l9 4v8l-9 4H17l-5-4v-8z"/>
    <path fill="#839fb2" d="M39 10l7 4v6l-7 4 2-7zM17 11l-2 4v4l2 4 2-6z"/>
    <path fill="#fff2c8" d="M55 9l4 2v4h-4zm0 10h4v4l-4 2z"/>
    <path fill="#c74955" d="M5 10h4v4H5zm0 10h4v4H5z"/>
    <path class="fisher-police-red" d="M25 8h5v9h-5z"/>
    <path class="fisher-police-blue" d="M25 17h5v9h-5z"/>
  </svg>`;
  const officer = `<svg viewBox="0 0 32 32" aria-hidden="true">
    <path fill="#0d203b" d="M7 8h12v5H7zM7 19h12v5H7z"/>
    <path fill="#2d557d" stroke="#96bad8" stroke-width="1" d="M8 10l6-5 8 3v16l-8 3-6-5z"/>
    <path fill="#e7b38c" d="M18 10h8l3 3v6l-3 3h-8z"/>
    <path fill="#172945" d="M20 8h6l4 5v6l-4 5h-6l3-8z"/>
    <path fill="#799cc1" d="M24 11h3l2 3v4l-2 3h-3l2-5z"/>
    <path fill="#f5cb72" d="M25 14l3 2-3 2z"/>
  </svg>`;

  function skin(markup) {
    return {
      tick(el, u, ctx) {
        if (!el.querySelector("svg")) el.innerHTML = markup;
        const angle = Math.atan2(-(u.vy || 0), u.vx || 0);
        el.style.setProperty("--police-angle", `${angle}rad`);
        el.dataset.policeFlash = String(Math.floor((ctx.time || 0) * 8) % 2);
      },
      unmount(el) { el.querySelector("svg")?.remove(); },
    };
  }

  window.lookFX = window.lookFX || {};
  window.lookFX["fisher-police-car"] = skin(car);
  window.lookFX["fisher-police-officer"] = skin(officer);

  // 结束或暂停时服务端仍会发同一快照，警笛和逮捕只播一次。
  const handled = new Set();
  function first(key) {
    if (handled.has(key)) return false;
    handled.add(key);
    return true;
  }

  arena.registerShot("electric", (fx, ctx) => {
    if (!first(`electric:${fx.unitId}:${ctx.time}`)) return;
    const el = arena.spawnFx("fx-fisher-electric", ctx.x, ctx.y, fx.kind, {
      "--electric-size": `${96 * ctx.scale}px`,
    });
    if (el) el.innerHTML = `<svg viewBox="0 0 100 100" aria-hidden="true">
      <circle cx="50" cy="50" r="37" fill="none" stroke="#70e6ff" stroke-width="2" stroke-dasharray="6 5"/>
      <path fill="none" stroke="#b5f4ff" stroke-width="2.5" d="M15 32l10 3-3 8 12 5M72 19l-1 13-8 2 6 9M83 68l-11-3-2 10-13 4M24 74l7-7-5-7 11-3"/>
    </svg>`;
  });

  arena.registerShot("police-siren", (fx) => {
    if (!first(`siren:${fx.unitId}`)) return;
    const base = arena.lookOf(fx.kind).base;
    const voice = arena.playAudio(`${base}/fx/police-siren.wav`, { mode: arena.SOUND_MODE.IMMUNE, volume: 0.74 });
    voice.audio.addEventListener("ended", () => voice.dispose(), { once: true });
  });

  arena.registerShot("arrest", (fx, ctx) => {
    if (!first(`arrest:${fx.unitId}`)) return;
    const el = arena.spawnFx("fx-fisher-arrest", ctx.x, ctx.y, fx.kind, {
      "--arrest-size": `${fx.amount * 2 * ctx.scale}px`,
      "--arrest-x": `${fx.vx * ctx.scale}px`,
      "--arrest-y": `${-fx.vy * ctx.scale}px`,
      "--arrest-facing": `${Math.atan2(fx.vy, -fx.vx)}rad`,
    }, document.getElementById("over"));
    if (!el) return;
    el.dataset.exportTail = "";
    el.innerHTML = `<i class="fisher-arrest-light"></i><i class="fisher-arrest-target"></i>
      <i class="fisher-arrest-officer">${officer}</i>
      <svg class="fisher-arrest-cuffs" viewBox="0 0 40 24" aria-hidden="true">
        <path d="M15 12h10" stroke="currentColor" stroke-width="3"/>
        <circle cx="9" cy="12" r="6" fill="none" stroke="currentColor" stroke-width="3"/>
        <circle cx="31" cy="12" r="6" fill="none" stroke="currentColor" stroke-width="3"/>
      </svg><b class="fisher-arrest-label">已逮捕</b>`;
  });
})();
