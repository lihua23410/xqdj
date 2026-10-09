window.lookFX = window.lookFX || {};
window.maggotFlyCD = window.maggotFlyCD || {};
window.maggotStenchAt = 0;

function setStench(on, layers) {
  // 挂在 #fx：被场地 clip-path 裁切，且在单位层之下，不会盖住球/血条。
  const root = document.getElementById("fx");
  if (!root) return;
  let el = root.querySelector(":scope > .maggot-stench");
  if (!on) {
    if (el) el.remove();
    return;
  }
  if (!el) {
    el = document.createElement("div");
    el.className = "maggot-stench";
    for (let i = 0; i < 5; i++) {
      el.appendChild(document.createElement("i"));
    }
    root.appendChild(el);
  }
  el.style.setProperty("--stench", String(Math.max(1, layers || 1)));
}

function expireStench() {
  if (!window.maggotStenchAt) return;
  if (performance.now() - window.maggotStenchAt > 280) {
    window.maggotStenchAt = 0;
    setStench(false);
  }
}

function maggotHP(el, u) {
  if (!el || !u) return;
  let tag = el.querySelector(":scope > .maggot-hp");
  if (!tag) {
    tag = document.createElement("span");
    tag.className = "maggot-hp";
    el.appendChild(tag);
  }
  tag.textContent = String(Math.round(u.hp));
}

function maggotUnmount(el) {
  el?.querySelector(":scope > .maggot-hp")?.remove();
  el?.querySelector(":scope > .maggot-body")?.remove();
  el?.querySelector(":scope > .fly-body")?.remove();
}

function ensureBabyBody(el) {
  let body = el.querySelector(":scope > .maggot-body");
  if (body) return body;
  body = document.createElement("div");
  body.className = "maggot-body";
  const worm = document.createElement("div");
  worm.className = "maggot-worm";
  for (let i = 0; i < 4; i++) {
    const seg = document.createElement("i");
    seg.className = "maggot-seg";
    worm.appendChild(seg);
  }
  const head = document.createElement("b");
  head.className = "maggot-head";
  worm.appendChild(head);
  body.appendChild(worm);
  el.appendChild(body);
  return body;
}

function ensureFlyBody(el) {
  let body = el.querySelector(":scope > .fly-body");
  if (body) return body;
  body = document.createElement("div");
  body.className = "fly-body";

  const wingL = document.createElement("i");
  wingL.className = "fly-wing left";
  const wingR = document.createElement("i");
  wingR.className = "fly-wing right";

  const abdomen = document.createElement("b");
  abdomen.className = "fly-abdomen";
  for (let i = 0; i < 3; i++) {
    abdomen.appendChild(document.createElement("em"));
  }

  const thorax = document.createElement("b");
  thorax.className = "fly-thorax";

  const head = document.createElement("b");
  head.className = "fly-head";
  const eyes = document.createElement("span");
  eyes.className = "maggot-eyes";
  eyes.appendChild(document.createElement("i"));
  eyes.appendChild(document.createElement("i"));
  head.appendChild(eyes);

  const legs = document.createElement("span");
  legs.className = "fly-legs";
  for (let i = 0; i < 6; i++) {
    legs.appendChild(document.createElement("i"));
  }

  body.appendChild(wingL);
  body.appendChild(wingR);
  body.appendChild(abdomen);
  body.appendChild(thorax);
  body.appendChild(head);
  body.appendChild(legs);
  el.appendChild(body);
  return body;
}

window.lookFX.maggot = { tick() {}, unmount() {} };
window.lookFX["maggot-baby"] = {
  tick(el, u) {
    if (!el || !u) return;
    maggotHP(el, u);
    const body = ensureBabyBody(el);
    const ang = Math.atan2(-(u.vy || 0), u.vx || 1);
    body.style.setProperty("--ang", `${ang}rad`);
    const sp = Math.hypot(u.vx || 0, u.vy || 0);
    el.classList.toggle("is-crawl", sp > 40);
    expireStench();
  },
  unmount: maggotUnmount,
};
window.lookFX["maggot-fly"] = {
  tick(el, u) {
    if (!el || !u) return;
    maggotHP(el, u);
    const body = ensureFlyBody(el);
    const ang = Math.atan2(-(u.vy || 0), u.vx || 1);
    body.style.setProperty("--ang", `${ang}rad`);
    const sp = Math.hypot(u.vx || 0, u.vy || 0);
    el.classList.toggle("is-buzz", sp > 60);
    const cd = !!window.maggotFlyCD[u.id];
    el.classList.toggle("is-breed-cd", cd);
    el.classList.toggle("is-ready", !cd);
    expireStench();
  },
  unmount(el) {
    maggotUnmount(el);
    el?.classList.remove("is-breed-cd", "is-ready", "is-buzz");
    setTimeout(() => {
      if (document.querySelectorAll(".ball.look-maggot-fly").length < 3) {
        setStench(false);
      }
    }, 0);
  },
};
window.lookFX["maggot-egg"] = {
  tick(el, u) {
    maggotHP(el, u);
    expireStench();
  },
  unmount: maggotUnmount,
};

arena.registerShot("fly-overlap", (fx, ctx) => {
  arena.spawnFx("fx-fly-overlap", ctx.x, ctx.y, ctx.kind);
});

arena.registerShot("fly-egg", (fx, ctx) => {
  arena.spawnFx("fx-fly-egg", ctx.x, ctx.y, ctx.kind);
});

arena.registerShot("fly-eye", (fx) => {
  if (!fx || fx.unitId == null) return;
  window.maggotFlyCD[fx.unitId] = (fx.amount || 0) > 0;
});

arena.registerShot("spirit", (fx, ctx) => {
  const el = arena.spawnFx("fx-spirit", ctx.x, ctx.y - 18, fx.kind);
  if (el) el.textContent = "抖擞精神";
});

arena.registerShot("stench", (fx) => {
  const n = fx && fx.amount ? fx.amount : 0;
  window.maggotStenchAt = performance.now();
  if (n >= 3) {
    setStench(true, Math.floor(n / 3));
  } else {
    setStench(false);
  }
});
