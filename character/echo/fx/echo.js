// 回声的声呐演出。
//
// 碎片每碰到敌人一次，在它身上补一格扫描；碎片完全不出伤。
// 10 格满后冻结为 6 秒锁定，期间不再补层或逐层衰减；结束清零，再回到扫描。
// 未满时每层独立存活 10 秒；锁定时十格保持全亮，到期一起收起。
//
// 扫描环挂在回声自己的 lookFX 上，每帧读 ctx.units 里敌人的 marks 来画；
// 这样不用改 app.js，也能在回声死后靠 until 定时器把环收掉。
window.lookFX = window.lookFX || {};

(function () {
  const A = window.arena;
  if (!A) return;

  const base = document.currentScript?.dataset.pack || "/ball/回声";
  const modes = A.SOUND_MODE;
  const sounds = {};
  const definitions = {
    scan: { mode: modes.NORMAL, priority: 210, volume: 0.65 },
    fire: { mode: modes.NORMAL, priority: 205, volume: 0.65, voices: 2 },
    burst: { mode: modes.NORMAL, priority: 205, volume: 0.6, voices: 2, gap: 0.08 },
    ping: { mode: modes.IMMUNE, volume: 0.55, voices: 3, gap: 0.06 },
    lock: { mode: modes.IMMUNE, volume: 0.8, voices: 2, gap: 0.12 },
    aim: { mode: modes.IMMUNE, volume: 0.45, voices: 2 },
    ram: { mode: modes.PREEMPT, priority: 130, volume: 0.85, voices: 2 },
    wall: { mode: modes.IMMUNE, volume: 0.55, voices: 2, gap: 0.08 },
    bounce: { mode: modes.NORMAL, priority: 220, volume: 0.3, voices: 2, gap: 0.09 },
    fade: { mode: modes.NORMAL, priority: 220, volume: 0.3, gap: 0.15 },
  };
  for (const [name, options] of Object.entries(definitions)) {
    const pool = Array.from({ length: options.voices || 1 }, () => {
      const audio = new Audio(base + "/fx/" + name + ".wav");
      audio.preload = "auto";
      return A.voice(audio, options);
    });
    sounds[name] = { pool, next: 0, last: -Infinity, gap: options.gap || 0, volume: options.volume };
  }

  function playSound(name, ctx, amount = 1) {
    const sound = sounds[name];
    const now = Number.isFinite(ctx?.time) ? ctx.time : performance.now() / 1000;
    const elapsed = now - sound.last;
    // Match time gives live play and exports the same limit; a new match may reset it.
    if (elapsed >= 0 && elapsed + 1e-9 < sound.gap) return;
    sound.last = now;
    const voice = sound.pool[sound.next++ % sound.pool.length];
    voice.stop();
    voice.reset();
    voice.audio.volume = sound.volume * Math.max(0, Math.min(1, amount));
    voice.play();
  }

  for (const [event, sound] of Object.entries({
    "echo-scan": "scan", "echo-fire": "fire", "echo-found": "lock",
    "echo-ram": "ram", "echo-wall": "wall",
    "echo-bounce": "bounce", "echo-fade": "fade",
  })) {
    A.registerShot(event, (fx, ctx) => playSound(sound, ctx));
  }

  const rings = new Map(); // "owner:target" -> { el, owner, target, timer }
  let host = null;

  function ensureHost(root) {
    if (!root) return null;
    if (!host || !host.isConnected || host.parentElement !== root) {
      host = document.createElement("div");
      host.className = "fx echo-scans";
      root.appendChild(host);
    }
    return host;
  }

  function keyOf(owner, target) {
    return owner + ":" + target;
  }

  function drop(key) {
    const ring = rings.get(key);
    if (!ring) return;
    if (ring.timer) clearTimeout(ring.timer);
    ring.el.remove();
    rings.delete(key);
  }

  function makeRing(key, owner, target) {
    const el = document.createElement("div");
    el.className = "echo-scan";
    el.innerHTML = '<i class="echo-scan-fill"></i><i class="echo-scan-dash"></i>';
    host.appendChild(el);
    const ring = { el, owner, target, timer: 0, zeroAt: 0, prev: 0, lastAt: 0, lockUntil: 0 };
    rings.set(key, ring);
    return ring;
  }

  function place(ring, o, ctx) {
    const p = A.screenPos(o.x, o.y, ctx.scale, ctx.cx, ctx.cy);
    const size = Math.max(44, (o.radius || 18) * ctx.scale * 2 + 24);
    ring.el.style.width = size + "px";
    ring.el.style.height = size + "px";
    ring.el.style.left = p[0] + "px";
    ring.el.style.top = p[1] + "px";
  }

  // 每一层独立活 lifetime 秒；用最新一次命中 + lifetime 作为 mark 归零的兜底时间，
  // 回声死后定时器也能把环收掉。
  function armTimer(ring, key, mark, ctx, lockUntil) {
    if (!(mark.lifetime > 0)) return;
    const zero = lockUntil > 0 ? lockUntil : (ring.lastAt || ctx.time) + mark.lifetime;
    if (ring.zeroAt === zero) return;
    ring.zeroAt = zero;
    if (ring.timer) clearTimeout(ring.timer);
    const ms = Math.max(0, (zero - ctx.time) * 1000) + 80;
    const self = ring;
    ring.timer = setTimeout(function () {
      if (rings.get(key) === self) drop(key);
    }, ms);
  }

  window.lookFX.echo = {
    unmount(el) {
      const owner = el && el.dataset ? el.dataset.echoOwner : "";
      for (const [key, ring] of Array.from(rings)) {
        if (!owner || String(ring.owner) === owner) drop(key);
      }
      if (!rings.size && host) {
        host.remove();
        host = null;
      }
      if (el) el.classList.remove("echo-locked");
    },

    tick(el, u, ctx) {
      if (!el || !el.classList.contains("look-echo")) return;
      el.dataset.echoOwner = String(u.id);
      const root = ensureHost(ctx && ctx.fxRoot);
      if (!root) return;

      const kind = "回声计数" + u.id;
      const freezeKind = kind + "锁定";
      const seen = new Set();
      let locked = false;

      for (const o of (ctx && ctx.units) || []) {
        if (o.slot === u.slot) continue;
        const frozen = (o.marks || []).find((m) => m.kind === freezeKind && m.stacks > 0);
        const mark = frozen || (o.marks || []).find((m) => m.kind === kind);
        if (!mark || mark.stacks <= 0) continue;

        const key = keyOf(u.id, o.id);
        seen.add(key);
        let ring = rings.get(key);
        if (!ring) ring = makeRing(key, u.id, o.id);
        place(ring, o, ctx);

        // 锁定到期时，不把清理指令尚未落地的迟到扫描层画成下一轮。
        if (!frozen && ring.lockUntil > 0 && ctx.time >= ring.lockUntil &&
            mark.nextDecayAt > 0 && mark.nextDecayAt - mark.lifetime < ring.lockUntil - 1e-9) {
          ring.el.style.display = "none";
          continue;
        }
        ring.el.style.display = "";
        if (frozen) ring.lockUntil = frozen.nextDecayAt || 0;

        const full = mark.stacks >= 10;
        locked = locked || full;
        if (mark.stacks > ring.prev) ring.lastAt = ctx.time;
        if (!full && !ring.lockUntil && mark.stacks < ring.prev) playSound("fade", ctx);
        ring.prev = mark.stacks;
        ring.el.style.setProperty(
          "--scan",
          ((Math.min(10, mark.stacks) / 10) * 360).toFixed(0) + "deg",
        );
        const left =
          !full && mark.lifetime > 0 && mark.nextDecayAt > 0
            ? Math.max(0, (mark.nextDecayAt - ctx.time) / mark.lifetime)
            : 1;
        ring.el.style.setProperty("--left", Math.min(1, left).toFixed(3));
        ring.el.classList.toggle("locked", full);
        armTimer(ring, key, mark, ctx, frozen ? ring.lockUntil : 0);
      }

      for (const key of Array.from(rings.keys())) {
        const ring = rings.get(key);
        if (ring && ring.owner === u.id && !seen.has(key)) {
          if (!ring.lockUntil && ring.prev > 0) playSound("fade", ctx);
          drop(key);
        }
      }

      el.classList.toggle("echo-locked", locked);
    },
  };

  // 碎片命中：落点一圈声呐；第 10 格那一圈变白。
  A.registerShot("echo-ping", function (fx, ctx) {
    // The authoritative six-second window emits echo-found; the tenth ping stays visual.
    if ((fx.amount || 0) < 10) playSound("ping", ctx, 0.6 + Math.max(0, fx.amount || 0) * 0.04);
    const el = A.spawnFx("fx-echo-ping", ctx.x, ctx.y, ctx.kind);
    if (el) el.classList.toggle("locked", (fx.amount || 0) >= 10);
  });

  // 星弹炸开：金色大圈。
  A.registerShot("echo-burst", function (fx, ctx) {
    playSound("burst", ctx);
    A.spawnFx("fx-echo-burst", ctx.x, ctx.y, ctx.kind);
  });

  // 锁定：一圈收束，表示“找到了”。
  A.registerShot("echo-lock", function (fx, ctx) {
    playSound("aim", ctx);
    A.spawnFx("fx-echo-lock", ctx.x, ctx.y, ctx.kind);
  });

  // 突进：一道青色冲刺痕。
  A.registerShot("echo-dash", function (fx, ctx) {
    A.spawnFx("fx-echo-dash", ctx.x, ctx.y, ctx.kind);
  });
})();
