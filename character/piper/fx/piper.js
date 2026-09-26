window.lookFX = window.lookFX || {};
window.piperLoot = window.piperLoot || {};

// 赃物计数：本体球外一圈小金点，攒着等下一次指挥（指挥会把它吃掉换成咬伤加成）。
function pips(el, n) {
  let row = el.querySelector(":scope > .piper-loot");
  if (!row) {
    row = document.createElement("span");
    row.className = "piper-loot";
    for (let i = 0; i < 3; i++) {
      const pip = document.createElement("i");
      const ang = (i / 3) * Math.PI * 2 - Math.PI / 2;
      pip.style.left = `${50 + Math.cos(ang) * 82}%`;
      pip.style.top = `${50 + Math.sin(ang) * 82}%`;
      row.appendChild(pip);
    }
    el.appendChild(row);
  }
  const slots = row.querySelectorAll("i");
  slots.forEach((pip, i) => pip.classList.toggle("full", i < n));
}

window.lookFX.piper = {
  unmount(el) {
    const id = el && el.dataset.piperId;
    if (id) delete window.piperLoot[id];
    el?.querySelector(":scope > .piper-loot")?.remove();
  },
  tick(el, u) {
    if (!el || !u) return;
    el.dataset.piperId = String(u.id);
    const n = window.piperLoot[u.id] || 0;
    pips(el, n);
  },
};

window.lookFX.rat = { tick() {}, unmount() {} };

// 赃物数是状态通道，不是一次性特效：只更新计数，不画东西。
arena.registerShot("loot", (fx) => {
  if (!fx || !fx.unitId) return;
  window.piperLoot[fx.unitId] = fx.amount || 0;
});
