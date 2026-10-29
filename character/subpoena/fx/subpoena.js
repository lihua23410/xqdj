// 传票员：票面数字和纸票演出。
//
// Go 那边每帧发一次 FX{name:"ticket", amount:票面}；这里按单位 id 记一张表，
// 皮肤 tick 每帧读表把数字刷到纸角上方。票被撞掉后单位消失，表里那条没人再读，
// shot.js 在表长过一定条数时整表清一次，下一帧的票面数字会自己补回来。
window.lookFX = window.lookFX || {};
window.subpoenaDmg = window.subpoenaDmg || {};

(function () {
  const A = window.arena;
  if (!A) return;

  function fmtAmount(v) {
    const n = Number(v) || 0;
    if (Number.isInteger(n)) return String(n);
    return n.toFixed(1).replace(/\.0$/, "");
  }

  window.lookFX.ticket = {
    tick(el, u) {
      if (!el || !u) return;
      const dmg = window.subpoenaDmg[u.id];
      if (dmg == null) return;
      let label = el.querySelector(".ticket-amount");
      if (!label) {
        label = document.createElement("span");
        label.className = "ticket-amount";
        el.appendChild(label);
      }
      const text = fmtAmount(dmg);
      if (label.textContent !== text) label.textContent = text;
      // 票面越大纸越烫：给 CSS 一个 0–1 的数。
      el.style.setProperty("--heat", Math.min(1, Number(dmg) / 40).toFixed(2));
    },

    unmount(el) {
      const label = el && el.querySelector(".ticket-amount");
      if (label) label.remove();
      if (el) el.style.removeProperty("--heat");
    },
  };
})();
