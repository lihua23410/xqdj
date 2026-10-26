(function () {
  window.lookFX = window.lookFX || {};
  var S = (window.charcoalFX = window.charcoalFX || {});
  S.frames = S.frames || {};

  var RIG_HTML = [
    '<i class="charcoal-ground"></i>',
    '<div class="charcoal-sway">',
    '  <div class="charcoal-jitter">',
    '    <div class="charcoal-body">',
    '      <i class="charcoal-crust"></i>',
    '      <i class="charcoal-cracks"></i>',
    '      <i class="charcoal-mouth"><b></b></i>',
    '    </div>',
    '    <div class="charcoal-hat">',
    '      <i class="charcoal-hat-crown"></i>',
    '      <i class="charcoal-hat-brim"></i>',
    '    </div>',
    '    <i class="charcoal-sweat s1"></i>',
    '    <i class="charcoal-sweat s2"></i>',
    '    <div class="charcoal-smoke"><i></i><i></i><i></i><i></i><i></i><i></i></div>',
    '    <div class="charcoal-embers"><i></i><i></i><i></i><i></i><i></i><i></i><i></i><i></i></div>',
    '    <i class="charcoal-forge-flash"></i>',
    '  </div>',
    '</div>',
  ].join("");

  function mount(el) {
    if (!el) return null;
    var rig = el.querySelector(":scope > .charcoal-rig");
    if (!rig) {
      rig = document.createElement("div");
      rig.className = "charcoal-rig";
      rig.innerHTML = RIG_HTML;
      el.appendChild(rig);
    }
    return rig;
  }
  S.mount = mount;

  // 同一持有者身上的炭可能来自不止一只炭翁：按所有在跑的帧合计。
  S.countHeld = function (holderId) {
    var want = Number(holderId);
    var total = 0;
    var frames = S.frames || {};
    for (var key in frames) {
      var map = frames[key];
      if (!map || typeof map.get !== "function") continue;
      var arr = map.get(want);
      if (arr) total += arr.length;
    }
    return total;
  };

  S.selfCount = function (ownerId) {
    var map = (S.frames || {})[Number(ownerId)];
    if (!map || typeof map.get !== "function") return 0;
    var arr = map.get(Number(ownerId));
    return arr ? arr.length : 0;
  };

  function freshClass(rig, cls, ms) {
    rig.classList.remove(cls);
    void rig.offsetWidth;
    rig.classList.add(cls);
    clearTimeout(rig["_" + cls + "Timer"]);
    rig["_" + cls + "Timer"] = setTimeout(function () {
      rig.classList.remove(cls);
    }, ms);
  }

  S.flare = function (el) {
    var rig = mount(el);
    if (!rig) return;
    freshClass(rig, "is-forging", 700);
  };

  S.throwPose = function (el) {
    var rig = mount(el);
    if (!rig) return;
    freshClass(rig, "is-throwing", 520);
  };

  var lastStep = new WeakMap();

  window.lookFX.charcoal = {
    unmount: function (el) {
      if (!el) return;
      var rig = el.querySelector(":scope > .charcoal-rig");
      if (rig) rig.remove();
      el.removeAttribute("data-charcoal-load");
      el.style.removeProperty("--load");
      el.style.removeProperty("--r");
      el.style.removeProperty("--coal-count");
    },
    tick: function (el, u, ctx) {
      if (!el || !el.classList.contains("look-charcoal")) return;
      var rig = mount(el);
      var n = S.countHeld(u.id);
      if (el.dataset.charcoalLoad !== String(n)) {
        el.dataset.charcoalLoad = String(n);
        var load = Math.pow(Math.min(1, n / 6), 0.72);
        el.style.setProperty("--load", load.toFixed(3));
        el.style.setProperty("--coal-count", String(n));
        rig.classList.toggle("is-heavy", n >= 4);
        rig.classList.toggle("is-max", n >= 6);
      }
      var scale = (ctx && ctx.scale) || 1;
      var r = Math.max(9, (u.radius || 18) * scale);
      var wantR = r.toFixed(1) + "px";
      if (el.style.getPropertyValue("--r") !== wantR) {
        el.style.setProperty("--r", wantR);
      }

      // 挑着炭走路会带起灰；担子越沉，步子越碎、灰越大。
      var speed = Math.hypot(u.vx || 0, u.vy || 0);
      if (n <= 0 || speed < 24 || !ctx || !window.arena || !arena.screenPos) return;
      var now = ctx.now || performance.now();
      var last = lastStep.get(el) || 0;
      var gap = 380 + Math.min(6, n) * 45;
      if (now - last < gap) return;
      lastStep.set(el, now);
      var pos = arena.screenPos(u.x, u.y, scale, ctx.cx, ctx.cy);
      var dirX = speed > 1 ? u.vx / speed : 0;
      var load = Math.min(1, n / 6);
      var feet = r * (1 + load * 0.36) + 1.5;
      arena.spawnFx("charcoal-step", pos[0] - dirX * r * 0.35, pos[1] + feet, u.kind, {
        "--s": String(0.7 + Math.min(6, n) * 0.12),
        "--dx": (-dirX * (3 + n * 0.7)).toFixed(1) + "px",
        "--dy": (-1.5 - n * 0.2).toFixed(1) + "px",
      });
    },
  };
})();
