window.lookFX = window.lookFX || {};
window.radarVetBeam = window.radarVetBeam || {};
window.radarVetMark = window.radarVetMark || {};
window.radarVetLock = window.radarVetLock || {};
window.radarVetGun = window.radarVetGun || {};

const PANEL_ANGLES = [-90, -30, 30, 90, 150, 210];

window.lookFX["radar-vet"] = {
  unmount(el) {
    if (!el) return;
    el.querySelector(":scope > .radar-vet-beam")?.remove();
    el.querySelector(":scope > .radar-vet-gun")?.remove();
    el.querySelector(":scope > .radar-vet-lock")?.remove();
    el.querySelector(":scope > .radar-vet-reticle")?.remove();
    el.querySelector(":scope > .radar-vet-rings")?.remove();
  },
  tick(el, u, ctx) {
    if (!el || !el.classList.contains("look-radar-vet")) return;
    if (!el.querySelector(":scope > .radar-vet-reticle")) {
      const retic = document.createElement("i");
      retic.className = "radar-vet-reticle";
      el.appendChild(retic);
      const rings = document.createElement("i");
      rings.className = "radar-vet-rings";
      el.appendChild(rings);
    }
    const st = window.radarVetBeam[u.id];
    let beam = el.querySelector(":scope > .radar-vet-beam");
    if (!st || st.off) {
      beam?.remove();
    } else {
      if (!beam) {
        beam = document.createElement("i");
        beam.className = "radar-vet-beam";
        el.appendChild(beam);
      }
      const scale = (ctx && ctx.scale) || 1;
      const dx = st.dx || 0;
      const dy = st.dy || 0;
      const ang = Math.atan2(-dy, dx);
      beam.style.setProperty("--ang", `${ang}rad`);
      beam.style.setProperty("--len", `${Math.max(8, (st.len || 0) * scale)}px`);
    }
    paintLock(el, u, ctx);
    paintGun(el, u);
  },
};

function paintGun(el, u) {
  const st = window.radarVetGun && window.radarVetGun[u.id];
  let gun = el.querySelector(":scope > .radar-vet-gun");
  if (!st || st.off) {
    gun?.remove();
    return;
  }
  if (!gun) {
    gun = document.createElement("i");
    gun.className = "radar-vet-gun";
    gun.innerHTML = buildTurret();
    el.appendChild(gun);
  }
  const grow = Math.max(0, Math.min(1, st.grow || 0));
  const prev = gun._g == null ? grow : gun._g;
  const rising = grow + 1e-3 >= prev;
  gun._g = grow;
  const g = rising ? grow : Math.max(0, Math.ceil(grow * 4 - 1e-6) / 4);
  gun.classList.toggle("retract", !rising);
  const band = (a, b) => Math.max(0, Math.min(1, (g - a) / (b - a)));
  const punch = (t) => {
    const c1 = 1.7;
    const c3 = c1 + 1;
    return 1 + c3 * (t - 1) ** 3 + c1 * (t - 1) ** 2;
  };
  gun.style.setProperty("--panel", String(rising ? punch(band(0, 0.42)) : band(0, 0.3)));
  gun.style.setProperty("--core", String(rising ? punch(band(0.22, 0.62)) : band(0.3, 0.58)));
  gun.style.setProperty("--port", String(rising ? punch(band(0.48, 0.84)) : band(0.58, 0.84)));
  gun.style.setProperty("--glare", String(rising ? punch(band(0.72, 1)) : band(0.84, 1)));
  const kick = window.radarVetGunKick && window.radarVetGunKick[u.id];
  gun.classList.toggle("kick", !!(kick && performance.now() - kick < 90));
}

function buildTurret() {
  const parts = [
    '<i class="tur-ring"></i>',
    '<i class="tur-core"></i>',
    '<i class="tur-port"></i>',
    '<i class="tur-glare"></i>',
  ];
  for (const deg of PANEL_ANGLES) {
    parts.push(`<i class="tur-panel" style="--deg:${deg}deg"></i>`);
  }
  return parts.join("");
}

function paintLock(el, u, ctx) {
  const st = window.radarVetLock && window.radarVetLock[u.id];
  let lock = el.querySelector(":scope > .radar-vet-lock");
  if (!st || st.off) {
    lock?.remove();
    return;
  }
  if (!lock) {
    lock = document.createElement("i");
    lock.className = "radar-vet-lock";
    lock.innerHTML = [
      '<i class="lock-ring"></i>',
      '<i class="lock-bracket" style="--b:0deg"></i>',
      '<i class="lock-bracket" style="--b:90deg"></i>',
      '<i class="lock-bracket" style="--b:180deg"></i>',
      '<i class="lock-bracket" style="--b:270deg"></i>',
    ].join("");
    el.appendChild(lock);
  }
  const scale = (ctx && ctx.scale) || 1;
  const dx = ((st.x || 0) - (u.x || 0)) * scale;
  const dy = -((st.y || 0) - (u.y || 0)) * scale;
  const d = Math.max(18, ((st.r || 18) + 8) * scale * 2);
  lock.style.setProperty("--x", `${dx}px`);
  lock.style.setProperty("--y", `${dy}px`);
  lock.style.setProperty("--d", `${d}px`);
}

function markFrac(st) {
  const dur = Math.max(1, st.dur || 1);
  const t = Math.max(0, Math.min(1, (performance.now() - st.born) / dur));
  let frac = t * t * (3 - 2 * t);
  if (frac < 0.04) frac = 0.04;
  return { t, frac };
}

function paintMarkSize(mark, st) {
  const { t, frac } = markFrac(st);
  const full = st.r || 4;
  const r = Math.max(3, full * frac * (st.scale || 1) * 2);
  mark.style.setProperty("--r", `${r}px`);
  return t;
}

function finishMark(id, st) {
  if (!st || st.done) return;
  st.done = true;
  st.el?.remove();
  st.el = null;
  if (typeof window.radarVetBoom === "function") window.radarVetBoom(id, st);
}

function tickShellMark(el, u, ctx, lookClass, markClass) {
  if (!el || !el.classList.contains(lookClass)) return;
  const st = window.radarVetMark[u.id];
  let mark = el.querySelector(`:scope > .${markClass}`);
  if (!st || !st.born || st.done) {
    mark?.remove();
    return;
  }
  if (ctx && ctx.scale) st.scale = ctx.scale;
  if (!mark) {
    mark = document.createElement("i");
    mark.className = markClass;
    el.appendChild(mark);
  }
  st.el = mark;
  if (paintMarkSize(mark, st) >= 1) finishMark(u.id, st);
}

(function radarVetMarkLoop() {
  const marks = window.radarVetMark || {};
  const now = performance.now();
  for (const [id, st] of Object.entries(marks)) {
    if (!st || !st.born || st.done) continue;
    let mark = st.el && st.el.isConnected ? st.el : null;
    if (!mark) {
      const host = document.getElementById(`u-${id}`);
      mark = host && host.querySelector(":scope > .radar-shell-mark, :scope > .radar-shell-mark-sm");
      if (mark) st.el = mark;
    }
    if (mark) {
      if (paintMarkSize(mark, st) >= 1) finishMark(id, st);
    } else if (now - st.born >= st.dur) {
      finishMark(id, st);
    }
  }
  requestAnimationFrame(radarVetMarkLoop);
})();

window.lookFX["radar-shell"] = {
  unmount(el) {
    if (!el) return;
    el.querySelector(":scope > .radar-shell-mark")?.remove();
  },
  tick(el, u, ctx) {
    tickShellMark(el, u, ctx, "look-radar-shell", "radar-shell-mark");
  },
};

window.lookFX["radar-shell-sm"] = {
  unmount(el) {
    if (!el) return;
    el.querySelector(":scope > .radar-shell-mark-sm")?.remove();
  },
  tick(el, u, ctx) {
    tickShellMark(el, u, ctx, "look-radar-shell-sm", "radar-shell-mark-sm");
  },
};
