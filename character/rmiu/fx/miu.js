window.lookFX = window.lookFX || {};

function ensureMiuHPNum(el) {
  let tag = el.querySelector(":scope > .miu-hp-num");
  if (tag) return tag;
  tag = document.createElement("span");
  tag.className = "miu-hp-num";
  el.appendChild(tag);
  return tag;
}

window.lookFX.miu = {
  unmount(el) {
    el?.querySelector(":scope > .miu-hp-num")?.remove();
  },
  tick(el, u) {
    if (!el || !u) return;
    // 本体（战斗机）由主程序画头顶数字 hp-float，这里只给缪随从补同款数字
    if (u.role === "fighter") {
      window.lookFX.miu.unmount(el);
      return;
    }
    const tag = ensureMiuHPNum(el);
    tag.textContent = String(Math.max(0, Math.round(u.hp || 0)));
  },
};
