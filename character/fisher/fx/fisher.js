window.lookFX = window.lookFX || {};
window.fisherHUD = window.fisherHUD || {};
window.fisherWeight = window.fisherWeight || {};
window.fisherCarry = window.fisherCarry || {};

function fisherKgText(n) {
  const v = Number(n);
  if (!Number.isFinite(v) || v <= 0) return "";
  return `${v.toFixed(2)}kg`;
}

function fisherEnsureKg(el) {
  let lab = el.querySelector(":scope > .fish-kg");
  if (!lab) {
    lab = document.createElement("b");
    lab.className = "fish-kg";
    el.appendChild(lab);
  }
  return lab;
}

function fisherPackBase(u) {
  const look = (window.arena && typeof window.arena.lookOf === "function" && window.arena.lookOf(u.kind)) || {};
  return look.base || "/ball/钓鱼佬";
}

function fisherFaceAng(u) {
  const dx = u && Number.isFinite(u.vx) ? u.vx : 1;
  const dy = u && Number.isFinite(u.vy) ? u.vy : 0;
  if (Math.hypot(dx, dy) < 1e-6) return 0;
  return Math.atan2(-dy, dx);
}

window.lookFX.fisher = {
  unmount(el) {
    if (!el) return;
    el.querySelector(":scope > .fisher-rod")?.remove();
    el.querySelector(":scope > .fisher-bobber")?.remove();
    el.querySelector(":scope > .fisher-ring")?.remove();
    el.querySelector(":scope > .fisher-hold")?.remove();
    el.querySelector(":scope > .fish-kg")?.remove();
    el.removeAttribute("data-fisher-rod");
    el.removeAttribute("data-fisher-cast");
    el.removeAttribute("data-fisher-ram");
    el.removeAttribute("data-fisher-haste");
    el.querySelector(":scope > .fisher-haste")?.remove();
    el.style.removeProperty("--fisher-prog");
    el.style.removeProperty("--r");
  },
  tick(el, u, ctx) {
    if (!el || !el.classList.contains("look-fisher")) return;
    const st = window.fisherHUD[u.id] || { misses: 0, prog: 0, flags: 0 };
    const flags = st.flags || 0;
    const rod = (flags & 1) !== 0;
    const cast = (flags & 2) !== 0;
    const ram = (flags & 4) !== 0;
    el.dataset.fisherRod = rod ? "1" : "0";
    el.dataset.fisherCast = cast ? "1" : "0";
    el.dataset.fisherRam = ram ? "1" : "0";
    const misses = Math.max(0, st.misses || 0);
    el.dataset.fisherHaste = String(misses);
    el.style.setProperty("--fisher-haste", String(misses));
    el.style.setProperty("--fisher-prog", String(Math.max(0, Math.min(1, st.prog || 0))));
    const scale = (ctx && ctx.scale) || 1;
    el.style.setProperty("--r", `${Math.max(10, (u.radius || 18) * scale)}px`);

    if (!el.querySelector(":scope > .fisher-rod")) {
      const rodEl = document.createElement("i");
      rodEl.className = "fisher-rod";
      el.appendChild(rodEl);
    }
    if (!el.querySelector(":scope > .fisher-bobber")) {
      const bob = document.createElement("i");
      bob.className = "fisher-bobber";
      el.appendChild(bob);
    }
    if (!el.querySelector(":scope > .fisher-ring")) {
      const ring = document.createElement("i");
      ring.className = "fisher-ring";
      el.appendChild(ring);
    }
    let haste = el.querySelector(":scope > .fisher-haste");
    if (misses > 0) {
      if (!haste) {
        haste = document.createElement("i");
        haste.className = "fisher-haste";
        el.appendChild(haste);
      }
      const now = (ctx && ctx.now) || performance.now();
      if (now - (haste._last || 0) > 70) {
        haste._last = now;
        const n = Math.min(4, 1 + misses);
        for (let i = 0; i < n; i++) {
          const streak = document.createElement("i");
          streak.className = "fisher-streak";
          streak.style.setProperty("--s", String(0.6 + Math.random() * 0.8));
          streak.style.setProperty("--oy", `${(Math.random() - 0.5) * 18}px`);
          haste.appendChild(streak);
          streak.addEventListener("animationend", () => streak.remove());
        }
      }
    } else {
      haste?.remove();
    }

    const kg = ram ? fisherKgText(window.fisherCarry[u.id]) : "";
    let hold = el.querySelector(":scope > .fisher-hold");
    if (kg) {
      if (!hold) {
        hold = document.createElement("img");
        hold.className = "fisher-hold";
        hold.alt = "";
        hold.src = `${fisherPackBase(u)}/fish.png`;
        el.appendChild(hold);
      }
      hold.style.setProperty("--ang", `${fisherFaceAng(u) + Math.PI}rad`);
      fisherEnsureKg(el).textContent = kg;
    } else {
      hold?.remove();
      el.querySelector(":scope > .fish-kg")?.remove();
    }
  },
};

window.lookFX["fisher-pond"] = {
  unmount(el) {
    if (!el) return;
    el.querySelector(":scope > .pond-ripple")?.remove();
  },
  tick(el) {
    if (!el || !el.classList.contains("look-fisher-pond")) return;
    if (el.querySelector(":scope > .pond-ripple")) return;
    const rip = document.createElement("i");
    rip.className = "pond-ripple";
    el.appendChild(rip);
  },
};

window.lookFX["fisher-fish"] = {
  unmount(el) {
    if (!el) return;
    el.querySelector(":scope > .fisher-fish-art")?.remove();
    el.querySelector(":scope > .fish-kg")?.remove();
  },
  tick(el, u) {
    if (!el || !el.classList.contains("look-fisher-fish")) return;
    let art = el.querySelector(":scope > .fisher-fish-art");
    if (!art) {
      art = document.createElement("span");
      art.className = "fisher-fish-art";
      const img = document.createElement("img");
      img.alt = "";
      img.src = `${fisherPackBase(u)}/fish.png`;
      art.appendChild(img);
      el.appendChild(art);
    }
    art.style.setProperty("--ang", `${fisherFaceAng(u) + Math.PI}rad`);
    const kg = fisherKgText(window.fisherWeight[u.id]);
    const lab = fisherEnsureKg(el);
    lab.textContent = kg;
    lab.style.display = kg ? "" : "none";
  },
};
