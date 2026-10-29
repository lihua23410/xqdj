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
  const TEXT_FLOOR = 12;
  const NUM_MAX = 46;
  const NUM_MIN = 28;

  // 名字要整条可见：缩到 min 还放不下就接着缩，不用省略号兜底。
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
    let size = lo;
    el.style.fontSize = `${size}px`;
    while (el.scrollWidth > width && size > TEXT_FLOOR) {
      size -= 1;
      el.style.fontSize = `${size}px`;
    }
    return size;
  }

  const nameStretch = [1, 1];

  function fitNames() {
    const sizes = nameEls.map((el) => fitText(el, NAME_MAX, NAME_MIN));
    const tallest = Math.max(...sizes);
    nameEls.forEach((el, i) => {
      if (!el) return;
      const size = sizes[i];
      // 长名字缩了字号但纵向被压扁；只拉高、不拉宽，让两边字高一致。
      // 拉伸系数只记账，transform 交给片头统一合成（见 placeLine）。
      nameStretch[i] = size && tallest > size ? tallest / size : 1;
      el.style.transformOrigin = i === 0 ? "left bottom" : "right bottom";
      el.style.transform = nameStretch[i] > 1 ? `scaleY(${nameStretch[i].toFixed(3)})` : "none";
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
  // 服务端统一下发帧数账本，演出不占用正式比赛的假时钟。
  let TOTAL_INTRO, TOTAL_START, TOTAL_END, TOTAL_PROBE, FPS;
  let initial, probe, probeClockMark;
  let battlePrepared = false;

  let frames = [];
  let total = 0;
  let maxHp = [0, 0];
  let ready = false;
  let started = false;
  let endShown = false;
  let frameNo = 0;
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

  // ---- 片头 ----------------------------------------------------------
  // 0–1 秒逐个点出三行，1–4 秒移到顶栏，4–5 秒画面元素浮现。
  // 全部是帧号的纯函数：片头期间绝不推进假时钟，否则 setTimeout、角色 fx 时间轴
  // 和首拍的 CSS 动画都会跟着提前走，对局第一拍就会错位。
  const INTRO_APPEAR = 60; // 1.0 秒逐个点出
  const INTRO_MOVE = 180; // 3.0 秒移到顶栏
  const INTRO_REVEAL = 60; // 1.0 秒元素浮现
  const GAP = 104; // 三行行距
  const RISE = 12; // 入场时向上浮的像素
  const FADE = 15; // 单行淡入 0.25 秒
  const SHIFT = 36; // 换位 0.6 秒
  const BEATS = [0, 21, 42]; // 三关节拍：0.00 / 0.35 / 0.70 秒
  const INTRO_SCALE = 1.4; // 片头放大倍数（按全名可见收窄）

  const vsEl = document.getElementById("export-vs");
  const lineEls = [nameEls[0], vsEl, nameEls[1]];
  const lineStretch = [1, 1, 1];
  const lineGeom = [null, null, null];
  let frameCenter = { x: 540, y: 648 };
  let introDone = false;

  const clamp01 = (v) => (v < 0 ? 0 : v > 1 ? 1 : v);
  // GSAP power2.inOut 的同款曲线（三次缓入缓出）。
  const easeInOutCubic = (p) => (p < 0.5 ? 4 * p * p * p : 1 - Math.pow(-2 * p + 2, 3) / 2);
  const easeOutCubic = (p) => 1 - Math.pow(1 - p, 3);

  // 量一次就够：片头不动布局，只把“文字中心”和“这块框的位置”记下来。
  function measureLines() {
    const rect = frameEl.getBoundingClientRect();
    frameCenter = { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 };
    lineEls.forEach((el, i) => {
      if (!el) return;
      el.style.transform = "none"; // 量的是没拉伸过的框
      const box = el.getBoundingClientRect();
      const range = document.createRange();
      range.selectNodeContents(el);
      const text = range.getBoundingClientRect();
      // 缩放原点沿用到位后的那套：左边名字吊左边、右边名字吊右边、VS 居中。
      const ox = i === 0 ? 0 : i === 2 ? box.width : box.width / 2;
      const oy = i === 0 || i === 2 ? box.height : box.height / 2;
      el.style.transformOrigin = `${ox}px ${oy}px`;
      lineGeom[i] = {
        left: box.left,
        top: box.top,
        ox,
        oy,
        px: text.left + text.width / 2 - box.left,
        py: text.top + text.height / 2 - box.top,
        w: text.width,
      };
      if (i === 0 || i === 2) lineStretch[i] = nameStretch[i === 0 ? 0 : 1];
    });
  }

  // 三行围绕画布中线长大：一行压中线 → 两行关于中线对称 → VS 回到中线。
  // 行距跟着放大倍数走，否则 1.4× 的字高（约 106px）会压过 104 的间距。
  function introGap() {
    return GAP * Math.min(introScale(0), introScale(2), INTRO_SCALE);
  }

  // 换位默认 0.6 秒，但必须在下一拍（或移动开始）之前落位，
  // 否则出现段只有 1 秒时三行会停在半路、间距变成 135/109。
  function shiftSpan(from, until) {
    return Math.max(1, Math.min(SHIFT, until - from));
  }

  function lineOffset(i, f) {
    const gap = introGap();
    const span2 = shiftSpan(BEATS[2], INTRO_APPEAR);
    if (i === 0) {
      if (f < BEATS[1]) return 0;
      if (f < BEATS[2]) return (-gap / 2) * easeInOutCubic((f - BEATS[1]) / shiftSpan(BEATS[1], BEATS[2]));
      return -gap / 2 - (gap / 2) * easeInOutCubic((f - BEATS[2]) / span2);
    }
    if (i === 1) {
      if (f < BEATS[2]) return gap / 2;
      return (gap / 2) * (1 - easeInOutCubic((f - BEATS[2]) / span2));
    }
    return gap;
  }

  function lineOpacity(i, f) {
    return easeOutCubic(clamp01((f - BEATS[i]) / FADE));
  }

  // 全名优先：放大倍数先让位给“整条名字不出画”，再谈好看。
  function introScale(i) {
    const g = lineGeom[i];
    if (!g || !g.w) return INTRO_SCALE;
    return Math.min(INTRO_SCALE, 1000 / g.w);
  }

  // 到位后的自然位置：纵向拉伸过的名字按底边为原点长高，中心比原始中心高一点。
  function settledPoint(i) {
    const g = lineGeom[i];
    const sy = lineStretch[i];
    return { x: g.left + g.px, y: g.top + g.oy + sy * (g.py - g.oy), k: 1 };
  }

  function appearPoint(i) {
    return { x: frameCenter.x, y: frameCenter.y + lineOffset(i, INTRO_APPEAR), k: introScale(i) };
  }

  // 把第 i 行的“文字中心”摆到 (cx, cy)，并按 k 缩放；缩放不改变文字中心。
  function placeLine(i, cx, cy, k) {
    const el = lineEls[i];
    const g = lineGeom[i];
    if (!el || !g) return;
    const sy = k * lineStretch[i];
    const ax = g.left + g.ox + k * (g.px - g.ox);
    const ay = g.top + g.oy + sy * (g.py - g.oy);
    const dx = cx - ax;
    const dy = cy - ay;
    el.style.transform = `translate(${dx.toFixed(2)}px, ${dy.toFixed(2)}px) scale(${k.toFixed(4)}, ${sy.toFixed(4)})`;
  }

  function setHud(v) {
    root.style.setProperty("--intro-hud", String(v));
  }

  function applyIntro(f) {
    if (!lineGeom[0]) return;
    if (f < INTRO_APPEAR) {
      for (let i = 0; i < 3; i++) {
        const op = lineOpacity(i, f);
        lineEls[i].style.opacity = op.toFixed(3);
        placeLine(i, frameCenter.x, frameCenter.y + lineOffset(i, f) + RISE * (1 - op), introScale(i));
      }
      setHud(0);
      return;
    }
    const e = easeInOutCubic(clamp01((f - INTRO_APPEAR) / INTRO_MOVE));
    for (let i = 0; i < 3; i++) {
      const from = appearPoint(i);
      const to = settledPoint(i);
      lineEls[i].style.opacity = "1";
      placeLine(i, from.x + (to.x - from.x) * e, from.y + (to.y - from.y) * e, from.k + (to.k - from.k) * e);
    }
    setHud(easeOutCubic(clamp01((f - INTRO_APPEAR - INTRO_MOVE) / INTRO_REVEAL)));
  }

  // 片头收尾：把三行落回它们自己的自然状态（拉伸的名字恢复 scaleY，其余清空）。
  function finishIntro() {
    if (introDone) return;
    introDone = true;
    setHud(1);
    for (let i = 0; i < 3; i++) {
      const el = lineEls[i];
      if (!el) continue;
      el.style.opacity = "1";
      const sy = lineStretch[i];
      el.style.transform = i !== 1 && sy > 1 ? `scaleY(${sy.toFixed(3)})` : "none";
    }
  }

  async function boot() {
    const base = `/api/export/${encodeURIComponent(job)}`;
    const [res, metaRes] = await Promise.all([
      fetch(`${base}/frames`, { cache: "no-store" }),
      fetch(`${base}/meta`, { cache: "no-store" }),
    ]);
    if (!res.ok) throw new Error(await res.text());
    if (!metaRes.ok) throw new Error(await metaRes.text());
    const meta = await metaRes.json();
    const tl = meta.timeline;
    TOTAL_INTRO = tl.titleFrames; TOTAL_START = tl.holdStartFrames;
    TOTAL_END = tl.endFrames; TOTAL_PROBE = tl.probeFrames; FPS = tl.fps;
    const text = await res.text();
    frames = text.split("\n").filter(Boolean).map((line) => ({ ...JSON.parse(line), intro: null }));
    if (!frames.length) throw new Error("没有帧");
    probe = meta.intro;
    initial = { ...(meta.initial || frames[0]), intro: null, effects: [] };
    total = tl.totalFrames;
    if (total !== TOTAL_INTRO + TOTAL_PROBE + TOTAL_START + frames.length + TOTAL_END) throw new Error("录像与时间线不匹配");
    window.__exportTotal = total;

    const allPacks = mergePacks([initial, ...frames]);
    window.__exportSetState({ ...frames[0], packs: allPacks });
    ensurePacks();
    await window.__packsWait();
    window.__exportSetState(frames[0]);
    ensurePacks();
    if (document.fonts && document.fonts.ready) await document.fonts.ready;

    const wrap = document.querySelector("#arena .stage-wrap");
    if (wrap && arenaEl) arenaEl.appendChild(wrap);
    setupHud(initial);
    root.classList.remove("hidden");
    fitNames();
    fitHpNumbers();
    measureLines();

    clock.set(0);
    frameNo = TOTAL_INTRO + TOTAL_PROBE + TOTAL_START;
    if (TOTAL_PROBE) {
      audioActive = false;
      probeClockMark = clock.checkpoint();
      window.__exportRender(initial, { film: probe, elapsed: 0 });
    } else {
      audioActive = true;
      window.__exportRender(frames[0]);
      battlePrepared = true;
    }
    scaleStage();
    pauseAnimations(0);
    if (!TOTAL_PROBE) {
      clock.run();
      pauseAnimations(0);
      clock.flush(0);
      pauseAnimations(0);
    }
    updateHp(initial);

    await new Promise((r) => realRAF(r));
    await new Promise((r) => realRAF(r));
    if (!TOTAL_PROBE) clock.flush(0);
    pauseAnimations(0);
    scaleStage();
    applyIntro(0);
    ready = true;
    started = true;
    window.__exportReady = true;
  }

  window.__exportStep = async function (i) {
    if (!ready) throw new Error("导出页还没准备好");
    if (i < TOTAL_INTRO) {
      // 片头只算样式，不动假时钟：比赛的时间轴还停在 0。
      audioActive = false;
      applyIntro(i);
      await new Promise((r) => realRAF(r));
      return;
    }
    finishIntro();
    if (i < TOTAL_INTRO + TOTAL_PROBE) {
      audioActive = false;
      const elapsed = (i - TOTAL_INTRO) / FPS;
      window.__exportRender(initial, { film: probe, elapsed });
      pauseAnimations(elapsed * 1000);
      scaleStage();
      await new Promise((r) => realRAF(r));
      return;
    }
    const first = TOTAL_INTRO + TOTAL_PROBE + TOTAL_START;
    if (!battlePrepared) {
      clock.discardSince(probeClockMark);
      frameNo = first;
      audioActive = true;
      // 首拍的音效记在正式比赛开始处；画面在首帧停留段先准备好。
      window.__exportRender(initial);
      scaleStage();
      pauseAnimations(0);
      clock.run(); clock.flush(0); pauseAnimations(0);
      updateHp(initial);
      battlePrepared = true;
    }
    if (i < first) {
      audioActive = false;
      await new Promise((r) => realRAF(r));
      return;
    }
    if (i < first + frames.length) {
      const fi = i - first;
      if (fi === 0) {
        audioActive = true;
        frameNo = first;
        if (TOTAL_PROBE) {
          window.__exportRender(frames[0]);
          clock.run(); pauseAnimations(0); clock.flush(0); pauseAnimations(0);
          scaleStage(); updateHp(frames[0]);
          // 首次加入比赛特效会新建合成层，跨过一次实际绘制再截图。
          await new Promise((r) => realRAF(r));
        }
        await new Promise((r) => realRAF(r));
        return;
      }
      const t = (fi * 1000) / FPS;
      frameNo = i;
      audioActive = true;
      clock.set(t);
      clock.run();
      window.__exportRender(frames[fi]);
      clock.run();
      pauseAnimations(t);
      clock.flush(t);
      pauseAnimations(t);
      scaleStage();
      updateHp(frames[fi]);
      tickMedia();
    } else {
      const j = i - (first + frames.length);
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
