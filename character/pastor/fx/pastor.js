(function () {
  const RUNES = "哎呦我去巴哈丢雷老某吼";
  const ownersBySnapshot = new WeakMap();

  window.lookFX = window.lookFX || {};

  function runeChar(u) {
    const i = Math.round(u && u.hp) - 1;
    return RUNES[i] || "";
  }

  function ownerOf(u, units) {
    if (!units) return null;
    let owners = ownersBySnapshot.get(units);
    if (!owners) {
      owners = new Map();
      for (const unit of units) {
        if (unit?.kind === "老牧师") owners.set(unit.id, unit);
      }
      ownersBySnapshot.set(units, owners);
    }
    return owners.get(u.ownerId);
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
      let mark = el.pastorGlyphMark;
      if (!mark) {
        mark = document.createElement("i");
        mark.style.animationDelay = (-((u.id || 0) % 7) * 0.33) + "s";
        el.appendChild(mark);
        el.pastorGlyphMark = mark;
      }
      const ch = runeChar(u);
      if (mark.textContent !== ch) mark.textContent = ch;
      const diameter = Math.round(2 * u.radius * (ctx?.scale ?? 1));
      const fontSize = Math.max(20, diameter * 1.12) + "px";
      if (mark.style.fontSize !== fontSize) mark.style.fontSize = fontSize;
      const owner = ownerOf(u, ctx && ctx.units);
      const rot = owner ? Math.atan2(owner.x - u.x, u.y - owner.y) : 0;
      mark.style.transform = "translate(-50%, -50%) rotate(" + rot + "rad)";
    },
    unmount(el) {
      el.pastorGlyphMark?.remove();
      delete el.pastorGlyphMark;
    },
  };
})();
