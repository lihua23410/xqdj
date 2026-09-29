(function () {
  const RUNES = "哎呦我去巴哈丢雷老某吼";

  window.lookFX = window.lookFX || {};

  function runeChar(u) {
    const i = Math.round(u && u.hp) - 1;
    return RUNES[i] || "";
  }

  function ownerOf(u, units) {
    if (!u) return null;
    for (const o of units || []) {
      if (o && o.id === u.ownerId) return o;
    }
    return null;
  }

  window.lookFX.pastor = {
    tick(el, u) {
      const fx = window.pastorFX;
      if (!el || !u) return;
      const on = fx && fx.screaming && fx.screaming[u.id];
      el.classList.toggle("is-scream", !!on);
      if (fx && typeof fx.tickAudio === "function") fx.tickAudio();
    },
  };

  window.lookFX["pastor-glyph"] = {
    tick(el, u, ctx) {
      if (!el || !u) return;
      let mark = el.querySelector(":scope > i");
      if (!mark) {
        mark = document.createElement("i");
        el.appendChild(mark);
      }
      const ch = runeChar(u);
      if (mark.textContent !== ch) mark.textContent = ch;
      const size = Math.max(10, el.clientWidth * 0.72);
      mark.style.fontSize = size + "px";
      const owner = ownerOf(u, ctx && ctx.units);
      let rot = 0;
      if (owner && ctx && typeof ctx.scale === "number") {
        const [gx, gy] = arena.screenPos(u.x, u.y, ctx.scale, ctx.cx, ctx.cy);
        const [ox, oy] = arena.screenPos(owner.x, owner.y, ctx.scale, ctx.cx, ctx.cy);
        rot = Math.atan2(ox - gx, oy - gy);
      }
      mark.style.transform = "translate(-50%, -50%) rotate(" + rot + "rad)";
    },
    unmount(el) {
      el?.querySelector(":scope > i")?.remove();
    },
  };
})();
