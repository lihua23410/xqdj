// 靶子的伤害播报：Go 每 0.2 秒送一次 dps 数字，这里记下来，
// 由 lookFX 在球头顶画一只一直说话的气泡。
(function () {
  window.dummyDps = window.dummyDps || {};
  window.lookFX = window.lookFX || {};

  function num(v) {
    const n = Number(v);
    return Number.isFinite(n) && n > 0 ? n : 0;
  }

  function one(v) {
    return num(v).toFixed(1);
  }

  function whole(v) {
    return String(Math.round(num(v)));
  }

  function sayBox(el) {
    let box = el.querySelector(":scope > .dummy-say");
    if (box) return box;
    box = document.createElement("div");
    box.className = "dummy-say";
    const main = document.createElement("b");
    main.className = "dummy-say-main";
    const sub = document.createElement("span");
    sub.className = "dummy-say-sub";
    box.appendChild(main);
    box.appendChild(sub);
    el.appendChild(box);
    return box;
  }

  arena.registerShot("dps", (fx) => {
    if (!fx || fx.unitId == null) return;
    window.dummyDps[fx.unitId] = {
      dps: num(fx.amount),
      recent: num(fx.vx),
      total: num(fx.vy),
    };
  });

  window.lookFX["dummy-dps"] = {
    unmount(el) {
      el?.querySelector(":scope > .dummy-say")?.remove();
    },
    tick(el, u) {
      if (!el || !el.classList.contains("look-dummy-dps")) return;
      const st = window.dummyDps[u.id] || { dps: 0, recent: 0, total: 0 };
      const box = sayBox(el);
      const main = box.querySelector(":scope > .dummy-say-main");
      const sub = box.querySelector(":scope > .dummy-say-sub");
      if (!main || !sub) return;

      const line = `DPS ${one(st.dps)}`;
      const detail = `近 5 秒 ${one(st.recent)} · 累计 ${whole(st.total)}`;
      const stamp = `${line}|${detail}`;
      if (box.dataset.stamp === stamp) return;
      box.dataset.stamp = stamp;
      main.textContent = line;
      sub.textContent = detail;
      // 换字时轻轻弹一下，像刚说完一句。
      box.classList.remove("bump");
      void box.offsetWidth;
      box.classList.add("bump");
    },
  };
})();
