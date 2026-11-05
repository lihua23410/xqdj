(function () {
  // 幻役气绝：原地竖起「气绝」两字，竖排。
  arena.registerShot("fujihime-scream", (fx, ctx) => {
    const el = arena.spawnFx("fx-fujihime-scream", ctx.x, ctx.y - 18, fx.kind);
    if (el) el.textContent = "「气绝」";
  });

  // 藤色结付：飘一行层数，方便看清现在是第几层。
  arena.registerShot("fujihime-tose", (fx, ctx) => {
    const el = arena.spawnFx("fx-fujihime-tose", ctx.x, ctx.y - 16, fx.kind);
    if (el) el.textContent = `${Math.round(fx.amount || 0)}`;
  });
  // 幻役贴到人：一小团紫闪。
  arena.registerShot("fujihime-doll-hit", (fx, ctx) => {
    arena.spawnFx("fx-fujihime-hit", ctx.x, ctx.y, fx.kind);
  });
  // 额外血条（藤色池子）：挂在球自己的节点里，单位被删掉时跟着一起没。
  // 满格按 100 点画，超过就画满；右边数字是真实剩余。
  function shieldSpec(absorb) {
    const pool = Math.max(0, Number(absorb) || 0);
    if (pool <= 0) return { shown: false, pct: 0, text: "0" };
    return {
      shown: true,
      pct: Math.max(6, Math.min(100, pool)),
      text: String(Math.max(1, Math.round(pool))),
    };
  }

  function shieldBarOf(el) {
    return el ? el.querySelector(".fujihime-shield") : null;
  }

  function paintShield(el, u) {
    const spec = shieldSpec(u && u.absorb);
    let bar = shieldBarOf(el);
    if (!spec.shown) {
      if (bar) bar.remove();
      return;
    }
    if (!bar) {
      bar = document.createElement("div");
      bar.className = "fujihime-shield";
      bar.append(document.createElement("i"), document.createElement("b"));
      el.appendChild(bar);
    }
    bar.firstElementChild.style.setProperty("--fill", `${spec.pct.toFixed(1)}%`);
    bar.lastElementChild.textContent = spec.text;
  }

  // 本体和幻役共用同一套挂件，各自的 look 名都登记一次。
  window.lookFX = window.lookFX || {};
  const shieldHook = {
    tick(el, u) {
      paintShield(el, u);
    },
    unmount(el) {
      const bar = shieldBarOf(el);
      if (bar) bar.remove();
    },
  };
  window.lookFX.fujihime = shieldHook;
  window.lookFX["fujihime-doll"] = shieldHook;
})();
