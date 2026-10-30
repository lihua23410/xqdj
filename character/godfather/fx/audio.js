// 教父的私有音效：仅观察现有快照，不向后端添加事件或改变战斗逻辑。
(function () {
  const A = window.arena;
  const base = document.currentScript?.dataset.pack || "/ball/教父";
  const definitions = {
    arrival: { mode: A.SOUND_MODE.NORMAL, priority: 220, volume: 0.4, voices: 2, gap: 0.25 },
    slash: { mode: A.SOUND_MODE.PREEMPT, priority: 199, volume: 0.8, voices: 2, gap: 0.12 },
    sniper: { mode: A.SOUND_MODE.IMMUNE, volume: 0.8, voices: 2, gap: 0.07 },
    deal: { mode: A.SOUND_MODE.NORMAL, priority: 215, volume: 0.6, voices: 2, gap: 0.18 },
    retreat: { mode: A.SOUND_MODE.NORMAL, priority: 225, volume: 0.35, voices: 2, gap: 0.25 },
  };
  const sounds = {};
  for (const [name, options] of Object.entries(definitions)) {
    sounds[name] = {
      options, last: -Infinity, next: 0,
      pool: Array.from({ length: options.voices }, () => {
        const audio = new Audio(`${base}/fx/${name}.wav`);
        audio.preload = "auto";
        return A.voice(audio, options);
      }),
    };
  }
  const families = new Map();
  const kinds = new Set(["教父暗杀者", "教父狙击者", "教父贩卖者"]);
  let lastTime = -Infinity;

  function rewind() {
    families.clear();
    for (const sound of Object.values(sounds)) {
      sound.last = -Infinity;
      sound.next = 0;
      for (const voice of sound.pool) voice.stop();
    }
  }
  function play(name, now) {
    const sound = sounds[name];
    if (now - sound.last + 1e-9 < sound.options.gap) return;
    sound.last = now;
    const voice = sound.pool[sound.next++ % sound.pool.length];
    voice.stop();
    voice.reset();
    voice.play();
  }
  function added(current, previous) {
    return [...current].some(id => !previous.has(id));
  }

  window.godfatherSFX = {
    observe(owner, ctx) {
      const now = Number.isFinite(ctx.time) ? ctx.time : 0;
      if (now < lastTime) rewind();
      lastTime = now;
      const units = ctx.units || [];
      const alive = new Set(units.filter(u => u.kind === "教父").map(u => u.id));
      for (const id of families.keys()) if (!alive.has(id)) families.delete(id);
      const members = units.filter(u => u.ownerId === owner.id && kinds.has(u.kind));
      const memberIds = new Set(members.map(u => u.id));
      const ownedIds = kind => new Set(units.filter(u => u.ownerId === owner.id && u.kind === kind).map(u => u.id));
      const rounds = ownedIds("教父狙击弹");
      const drugs = ownedIds("教父药物");
      const signaled = new Set((ctx.effects || [])
        .filter(fx => fx.kind === "教父" && fx.name === "aim" && fx.amount < 0)
        .map(fx => fx.unitId));
      const departing = new Set();
      const dashes = new Map();
      for (const u of members) {
        const speed = Math.hypot(u.vx || 0, u.vy || 0);
        const aim = window.godfatherAim?.[u.id];
        if (aim?.leaving && !(aim.time > now + 0.001)) signaled.add(u.id);
        if (signaled.has(u.id) || speed > 680) departing.add(u.id);
        if (u.kind === "教父暗杀者") dashes.set(u.id, !departing.has(u.id) && speed > 400);
      }
      const previous = families.get(owner.id);
      // 第一份快照只建立基线；中途进入不会把在场随从和飞行中的弹当成新动作。
      if (previous) {
        if (added(memberIds, previous.members)) play("arrival", now);
        if (added(rounds, previous.rounds)) play("sniper", now);
        if (added(drugs, previous.drugs)) play("deal", now);
        for (const [id, dashing] of dashes) {
          if (dashing && previous.dashes.has(id) && !previous.dashes.get(id)) play("slash", now);
        }
        for (const id of signaled) {
          if (memberIds.has(id) && previous.members.has(id) && !previous.departing.has(id)) play("retreat", now);
        }
      }
      families.set(owner.id, { members: memberIds, rounds, drugs, dashes, departing });
    },
  };
})();
