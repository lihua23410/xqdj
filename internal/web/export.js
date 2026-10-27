// 导出页：把服务端记下的观众快照逐帧喂给观战页，时间由 export-clock.js 推进。
(function () {
  const params = new URLSearchParams(location.search);
  const job = params.get("export");
  if (!job) return;

  document.body.classList.add("export");
  window.__exportReady = false;
  window.__exportError = "";

  const root = document.getElementById("export-root");
  const frameEl = document.getElementById("export-frame");
  const arenaEl = document.getElementById("export-arena");
  const winnerEl = document.getElementById("export-winner");
  const winnerSmall = winnerEl.querySelector("small");
  const winnerBig = winnerEl.querySelector("b");
  const hpEls = [0, 1].map((slot) => document.querySelector(`.export-hp[data-slot="${slot}"]`));
  const nameEls = [document.getElementById("export-name-0"), document.getElementById("export-name-1")];

  const NAME_MAX = 76;
  const NAME_MIN = 30;
  const NUM_MAX = 46;
  const NUM_MIN = 28;

  function fitText(el, max, min) {
    if (!el) return 0;
    const width = el.clientWidth;
    if (!width) return 0;
    el.style.fontSize = `${max}px`;
    let lo = min;
    let hi = max;
    while (hi - lo > 1) {
      const mid = (lo + hi) >> 1;
      el.style.fontSize = `${mid}px`;
      if (el.scrollWidth <= width) lo = mid;
      else hi = mid;
    }
    el.style.fontSize = `${lo}px`;
    return lo;
  }

  function fitNames() {
    const sizes = nameEls.map((el) => fitText(el, NAME_MAX, NAME_MIN));
    const tallest = Math.max(...sizes);
    nameEls.forEach((el, i) => {
      if (!el) return;
      const size = sizes[i];
      if (!size || tallest <= size) {
        el.style.transform = "none";
        return;
      }
      // 长名字缩了字号但纵向被压扁；只拉高、不拉宽，让两边字高一致。
      el.style.transformOrigin = i === 0 ? "left bottom" : "right bottom";
      el.style.transform = `scaleY(${(tallest / size).toFixed(3)})`;
    });
  }

  function fitHpNumbers() {
    for (const slot of [0, 1]) {
      const el = hpEls[slot] && hpEls[slot].querySelector(".export-hp-num");
      if (!el) continue;
      const current = el.textContent;
      el.textContent = String(Math.round(maxHp[slot] || 0));
      fitText(el, NUM_MAX, NUM_MIN);
      el.textContent = current;
    }
  }

  const clock = window.__clock;
  const realRAF = window.__realRAF;
  const TOTAL_START = 60;
  const TOTAL_END = 120;
  const FPS = 60;

  let frames = [];
  let total = 0;
  let maxHp = [0, 0];
  let ready = false;
  let started = false;
  let endShown = false;
  let frameNo = TOTAL_START;
  let audioActive = false;
  const audioEvents = [];
  const playingMedia = new Set();
  const bornAt = new WeakMap();

  // 不真的出声，只记“哪一帧要播什么”。服务端按这张表混音。
  // 关键：play() 必须把 paused 伪装成 false，否则角色的 tick 会每帧
  // 都以为还没播、反复 play，导出音轨就会叠出“电音”。
  const media = HTMLMediaElement.prototype;
  media.play = function () {
    if (this.__exportPlaying) return Promise.resolve();
    if (audioActive) {
      const src = audioPath(this.currentSrc || this.src || "");
      if (src) {
        const offset = this.currentTime || 0;
        audioEvents.push({
          src,
          frame: frameNo,
          volume: this.muted ? 0 : this.volume,
          loop: !!this.loop,
          rate: this.playbackRate || 1,
          preservesPitch: this.preservesPitch !== false,
          offset,
        });
        this.__exportVoice = audioEvents.length - 1;
        this.__exportPlaying = true;
        this.__exportAnchorFrame = frameNo;
        this.__exportAnchorTime = offset;
        playingMedia.add(this);
        try {
          if (!Object.prototype.hasOwnProperty.call(this, "paused")) {
            Object.defineProperty(this, "paused", {
              configurable: true,
              get() { return !this.__exportPlaying; },
            });
          }
        } catch (e) { /* 个别浏览器不允许覆盖，退化为旧的 paused 行为 */ }
      }
    }
    return Promise.resolve();
  };
  media.pause = function () {
    if (audioActive && this.__exportPlaying) {
      audioEvents.push({ stop: this.__exportVoice, frame: frameNo });
      this.__exportPlaying = false;
      playingMedia.delete(this);
    }
    return Promise.resolve();
  };

  // 音频不会真的走时钟，currentTime 由导出帧号推进；这样暂停/恢复时
  // 混音里的 offset 才是角色实际听到的位置。
  function tickMedia() {
    for (const el of playingMedia) {
      const t = (el.__exportAnchorTime || 0) + (frameNo - el.__exportAnchorFrame) / FPS;
      const d = el.duration;
      if (Number.isFinite(d) && d > 0 && t >= d) {
        el.__exportPlaying = false;
        playingMedia.delete(el);
        try { el.currentTime = d; } catch (e) { /* ignore */ }
      } else {
        try { el.currentTime = t; } catch (e) { /* ignore */ }
      }
    }
  }

  function audioPath(raw) {
    if (!raw) return "";
    try {
      const u = new URL(raw, location.href);
      if (u.protocol !== "http:" && u.protocol !== "https:") return "";
      return u.pathname;
    } catch {
      return "";
    }
  }

  function lineupOf(snap, slot) {
    const r = snap.roster && snap.roster[slot];
    if (r && r.length) return r;
    const s = snap.slots && snap.slots[slot];
    if (Array.isArray(s)) return s;
    return typeof s === "string" && s ? [s] : [];
  }

  function nameOf(snap, slot) {
    return lineupOf(snap, slot).join(" + ");
  }

  function colorOf(snap, slot) {
    const kind = lineupOf(snap, slot)[0];
    return (snap.looks && snap.looks[kind] && snap.looks[kind].color) || (slot === 0 ? "#ff7a2f" : "#ffe14d");
  }

  function fighters(snap, slot) {
    return (snap.units || []).filter((u) => u.role === "fighter" && u.slot === slot);
  }

  function hpOf(snap, slot) {
    return fighters(snap, slot).reduce((sum, u) => sum + (u.hp || 0), 0);
  }

  function updateHp(snap) {
    for (const slot of [0, 1]) {
      const cur = hpOf(snap, slot);
      const max = maxHp[slot] || 0;
      const pct = max > 0 ? Math.max(0, Math.min(1, cur / max)) : 0;
      const el = hpEls[slot];
      if (!el) continue;
      el.querySelector(".export-hp-num").textContent = String(Math.round(cur));
      el.querySelector(".export-hp-bar i").style.width = `${pct * 100}%`;
      el.style.setProperty("--hp-color", colorOf(snap, slot));
    }
  }

  function setWinner(opacity) {
    if (winnerEl) winnerEl.style.opacity = String(opacity);
  }

  function pauseAnimations(t) {
    let anims;
    try {
      anims = document.getAnimations();
    } catch {
      return;
    }
    for (const a of anims) {
      let born = bornAt.get(a);
      if (born === undefined) {
        try {
          a.pause();
        } catch {}
        born = t;
        bornAt.set(a, born);
      }
      try {
        a.currentTime = Math.max(0, t - born);
      } catch {}
    }
  }

  function setupHud(first) {
    for (const slot of [0, 1]) {
      nameEls[slot].textContent = nameOf(first, slot);
      nameEls[slot].style.setProperty("--accent", colorOf(first, slot));
      hpEls[slot].style.setProperty("--hp-color", colorOf(first, slot));
    }
    for (const f of frames) {
      for (const slot of [0, 1]) {
        const list = fighters(f, slot);
        const cur = list.reduce((sum, u) => sum + (u.hp || 0), 0);
        const max = list.reduce((sum, u) => sum + (u.maxHp || 0), 0);
        if (cur > maxHp[slot]) maxHp[slot] = cur;
        if (max > maxHp[slot]) maxHp[slot] = max;
      }
    }
    updateHp(first);
    const last = frames[frames.length - 1];
    const winner = (last && last.winner) || "";
    winnerBig.textContent = winner;
    if (!winner) {
      winnerEl.style.display = "none";
    } else if (winner === "平局") {
      winnerSmall.style.display = "none";
    }
    setWinner(0);
  }

  function scaleStage() {
    const wrap = arenaEl.querySelector(".stage-wrap");
    if (!wrap) return;
    const w = wrap.offsetWidth || 560;
    const h = wrap.offsetHeight || 560;
    const s = Math.min(1000 / w, 1000 / h);
    wrap.style.transform = `scale(${s})`;
    wrap.style.left = `${(1000 - w * s) / 2}px`;
    wrap.style.top = `${(1000 - h * s) / 2}px`;
  }

  function mergePacks(list) {
    const all = {};
    for (const snap of list) {
      for (const [key, pack] of Object.entries(snap.packs || {})) {
        if (!all[key]) all[key] = { base: pack.base, files: new Set(), factions: pack.factions };
        for (const file of pack.files || []) all[key].files.add(file);
      }
    }
    for (const pack of Object.values(all)) pack.files = [...pack.files];
    return all;
  }

  async function boot() {
    const res = await fetch(`/api/export/${encodeURIComponent(job)}/frames`, { cache: "no-store" });
    if (!res.ok) throw new Error(await res.text());
    const text = await res.text();
    frames = text.split("\n").filter(Boolean).map((line) => JSON.parse(line));
    if (!frames.length) throw new Error("没有帧");
    total = TOTAL_START + frames.length + TOTAL_END;
    window.__exportTotal = total;

    const allPacks = mergePacks(frames);
    window.__exportSetState({ ...frames[0], packs: allPacks });
    ensurePacks();
    await window.__packsWait();
    window.__exportSetState(frames[0]);
    ensurePacks();
    if (document.fonts && document.fonts.ready) await document.fonts.ready;

    const wrap = document.querySelector("#arena .stage-wrap");
    if (wrap && arenaEl) arenaEl.appendChild(wrap);
    setupHud(frames[0]);
    root.classList.remove("hidden");
    fitNames();
    fitHpNumbers();

    clock.set(0);
    audioActive = true;
    frameNo = TOTAL_START;
    window.__exportRender(frames[0]);
    scaleStage();
    pauseAnimations(0);
    clock.run();
    pauseAnimations(0);
    clock.flush(0);
    pauseAnimations(0);
    updateHp(frames[0]);

    await new Promise((r) => realRAF(r));
    await new Promise((r) => realRAF(r));
    clock.flush(0);
    pauseAnimations(0);
    scaleStage();
    ready = true;
    started = true;
    window.__exportReady = true;
  }

  window.__exportStep = async function (i) {
    if (!ready) throw new Error("导出页还没准备好");
    if (i < TOTAL_START) return;
    if (i < TOTAL_START + frames.length) {
      const fi = i - TOTAL_START;
      if (fi === 0) {
        audioActive = true;
        frameNo = TOTAL_START;
        return;
      }
      const t = (fi * 1000) / FPS;
      clock.set(t);
      clock.run();
      frameNo = i;
      audioActive = true;
      window.__exportRender(frames[fi]);
      clock.run();
      pauseAnimations(t);
      clock.flush(t);
      pauseAnimations(t);
      scaleStage();
      updateHp(frames[fi]);
      tickMedia();
    } else {
      const j = i - (TOTAL_START + frames.length);
      endShown = true;
      setWinner(Math.min(1, j / 36));
    }
    await new Promise((r) => realRAF(r));
  };

  window.__exportAudio = function () {
    return JSON.stringify(audioEvents);
  };

  boot().catch((err) => {
    window.__exportError = String((err && err.stack) || err);
    console.error("export boot", err);
  });
})();
