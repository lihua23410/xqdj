window.lookFX = window.lookFX || {};
window.orcaForm = window.orcaForm || {};
window.orcaRice = window.orcaRice || {};

(function () {
  const arena = window.arena;
  const riceN = 7;

  function baseOf(kind) {
    const look = arena && arena.lookOf ? arena.lookOf(kind) : null;
    return (look && look.base) || "/ball/大肥鱼";
  }

  function faceAng(u) {
    const vx = u && Number.isFinite(u.vx) ? u.vx : 1;
    const vy = u && Number.isFinite(u.vy) ? u.vy : 0;
    if (Math.hypot(vx, vy) < 1e-6) return 0;
    return Math.atan2(-vy, vx);
  }

  function art(el) {
    let img = el.querySelector(":scope > .orca-art");
    if (!img) {
      img = document.createElement("img");
      img.className = "orca-art";
      img.alt = "";
      el.appendChild(img);
    }
    return img;
  }

  window.lookFX.orca = {
    unmount(el) {
      el?.querySelector(":scope > .orca-art")?.remove();
    },
    tick(el, u, ctx) {
      if (!el || !u) return;
      const img = art(el);
      const root = baseOf(u.kind);
      const phase = window.orcaForm[u.id] || 0;
      const whale = phase >= 1;
      const leap = phase === 3;
      const dive = phase === 2;
      const file = leap ? "fx/orca-02.png" : whale ? "fx/orca-05.png" : "fx/fish-02.png";
      const src = `${root}/${file}`;
      if (img.getAttribute("src") !== src) img.src = src;
      img.classList.toggle("maid", !whale);
      img.classList.toggle("whale", whale && !dive && !leap);
      img.classList.toggle("dive", dive);
      img.classList.toggle("leap", leap);
      if (!whale) {
        img.style.transform = "translate(-50%, -50%)";
        return;
      }
      const now = (ctx && ctx.now) || performance.now();
      const bob = dive ? 0 : Math.sin(now / 220) * 0.12;
      const sink = dive ? " translateY(18%)" : "";
      img.style.transform = `translate(-50%, -50%) rotate(${faceAng(u) + bob}rad)${sink}`;
    },
  };

  window.lookFX["orca-rice"] = {
    unmount(el) {
      el?.querySelector(":scope > .orca-art")?.remove();
    },
    tick(el, u) {
      if (!el || !u) return;
      const img = art(el);
      let n = window.orcaRice[u.id];
      if (!Number.isFinite(n)) n = 0;
      n = Math.max(0, Math.min(riceN - 1, Math.round(n)));
      const src = `${baseOf(u.kind)}/fx/rice-0${n + 1}.png`;
      if (img.getAttribute("src") !== src) img.src = src;
      img.style.transform = "translate(-50%, -50%)";
    },
  };

  arena.registerShot("form", (fx) => {
    if (!fx || fx.unitId == null) return;
    window.orcaForm[fx.unitId] = Math.round(fx.amount || 0);
  });
  arena.registerShot("rice", (fx) => {
    if (!fx || fx.unitId == null) return;
    window.orcaRice[fx.unitId] = fx.amount || 0;
  });
  arena.registerShot("leap", (fx, ctx) => {
    if (!arena || !arena.spawnFx) return;
    const el = arena.spawnFx("fx-orca-flash", ctx.x, ctx.y, fx.kind);
    if (!el) return;
    const img = document.createElement("img");
    img.alt = "";
    img.src = `${baseOf(fx.kind)}/fx/orca-02.png`;
    el.appendChild(img);
  });
  arena.registerShot("splash", (fx, ctx) => {
    if (!arena || !arena.spawnFx) return;
    arena.spawnFx("fx-orca-splash", ctx.x, ctx.y, fx.kind);
    arena.spawnFx("fx-orca-splash-2", ctx.x, ctx.y, fx.kind);
  });
})();
