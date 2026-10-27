// 时计皮肤：白盘 + 黑针（时针 + 分针）。
// 状态由 shot.js 的两个 FX 喂进来：clock（本体表针 + 本拍窗口）、moment（每个残影自己的针位）。
// 两根针都按**模拟时间**逐帧算角度（不是 CSS 过渡）：引擎停帧/暂停时快照的 time 不走，针也就不走。
window.lookFX = window.lookFX || {};
window.tokeiSt = window.tokeiSt || {};

// 当前模拟时刻。锤子的 nail.js 也是这么读全局 state 的（包脚本和 app.js 共享同一个全局作用域）。
function tokeiNow() {
  if (typeof state !== "undefined" && state && Number.isFinite(Number(state.time))) {
    return Number(state.time);
  }
  return 0;
}

function tokeiEnsure(el, cls) {
  let n = el.querySelector(`:scope > .${cls}`);
  if (!n) {
    n = document.createElement("div");
    n.className = cls;
    el.appendChild(n);
  }
  return n;
}

// tokeiFace 一次性搭出刻度盘与两根针：12 个刻度每 30°
function tokeiFace(el) {
  const face = tokeiEnsure(el, "tokei-face");
  if (face.dataset.built) return face;
  face.dataset.built = "1";
  for (let i = 0; i < 12; i++) {
    const t = document.createElement("i");
    t.className = "tick";
    t.style.transform = `rotate(${i * 30}deg)`;
    face.appendChild(t);
  }
  for (const cls of ["hand hour", "hand minute"]) {
    const h = document.createElement("i");
    h.className = cls;
    face.appendChild(h);
  }
  return face;
}

// tokeiAngles 两根针的角度（度；0 = 正上，顺时针为正）。
// 本体：时针在本拍窗口 [from,to] 里从当前读数扫到下一拍读数（+2h），分针一拍正好一圈（8s/r）；
//   两根针都在窗口末尾——也就是刻档那一瞬——刚好落在整点上（先推时间再留时刻）。
// 残影：针停在它刻下的读数上（分针指 12，读出来就是 2:00 / 4:00……）。
// 残影踩影后的静默窗口：盘面压暗，时针退到上一小时再拨回原读数，分针同步走满一圈。
// 满仓（12 档读数 = 12:00）：没有下一拍，针停在 12:00（与 0:00 重合）。
window.tokeiAngles = function (st, now) {
  const hours = Number(st && st.hours) || 0;
  const from = Number(st && st.from) || 0;
  const to = Number(st && st.to) || 0;
  const ghost = !!(st && st.ghost);
  const coolFrom = Number(st && st.coolFrom) || 0;
  const coolTo = Number(st && st.coolTo) || 0;
  let phase = 0;
  if (ghost && coolTo > coolFrom && Number(now) >= coolFrom && Number(now) < coolTo) {
    phase = (Number(now) - coolFrom) / (coolTo - coolFrom);
    phase = Math.min(1, Math.max(0, phase));
    const back = Math.max(0, hours - 1);
    // 时针从上一小时走回原时刻；分针在同一个窗口里走满一圈，当冷却进度看。
    return { hour: (back + phase) * 30, minute: phase * 360, cooling: true };
  }
  if (!ghost && hours < 12 && to > from) {
    phase = (Number(now) - from) / (to - from);
    phase = Math.min(1, Math.max(0, phase));
  }
  const step = ghost ? 0 : 2;
  return { hour: (hours + step * phase) * 30, minute: phase * 360, cooling: false };
};

function paintTokei(el, id, ghost) {
  const st = window.tokeiSt[id] || { hours: 0, full: false };
  const face = tokeiFace(el);
  face.classList.toggle("dust", !ghost && !!st.full);
  const a = window.tokeiAngles(st, tokeiNow());
  face.classList.toggle("cooling", !!a.cooling);
  const hour = face.querySelector(".hand.hour");
  const minute = face.querySelector(".hand.minute");
  if (hour) hour.style.transform = `rotate(${a.hour}deg)`;
  if (minute) minute.style.transform = `rotate(${a.minute}deg)`;
}

window.lookFX.tokei = {
  unmount(el) {
    el.querySelector(":scope > .tokei-face")?.remove();
  },
  tick(el, u) {
    paintTokei(el, u.id, false);
  },
};

window.lookFX["tokei-moment"] = {
  unmount(el) {
    el.querySelector(":scope > .tokei-face")?.remove();
  },
  tick(el, u) {
    paintTokei(el, u.id, true);
  },
};
