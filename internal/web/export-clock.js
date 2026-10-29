// 导出模式专用的假时钟。只替换时间源，不改页面的渲染逻辑。
// 只有 /?export=... 会走到这里；普通观战页原样使用浏览器时间。
(function () {
  const EXPORT = new URLSearchParams(location.search).has("export");
  if (!EXPORT) return;

  const realRAF = window.requestAnimationFrame.bind(window);
  const realSetTimeout = window.setTimeout.bind(window);
  const realClearTimeout = window.clearTimeout.bind(window);

  let now = 0;
  let nextId = 1;
  const timers = new Map();
  const rafs = [];

  function addTimer(fn, delay, repeat, args) {
    if (typeof fn !== "function") return 0;
    const id = nextId++;
    const wait = Number.isFinite(delay) && delay > 0 ? delay : 0;
    timers.set(id, {
      at: now + wait,
      fn,
      args,
      repeat: repeat ? Math.max(1, wait) : 0,
    });
    return id;
  }

  window.setTimeout = (fn, delay, ...args) => addTimer(fn, delay, 0, args);
  window.setInterval = (fn, delay, ...args) => addTimer(fn, delay, 1, args);
  window.clearTimeout = (id) => timers.delete(id);
  window.clearInterval = (id) => timers.delete(id);

  window.requestAnimationFrame = (fn) => {
    if (typeof fn !== "function") return 0;
    rafs.push(fn);
    return nextId++;
  };
  window.cancelAnimationFrame = () => {};

  function setNow(ms) {
    if (Number.isFinite(ms) && ms > now) now = ms;
  }

  function runTimers() {
    let guard = 0;
    while (timers.size && guard++ < 200000) {
      let pickId = 0;
      let pick = null;
      for (const [id, t] of timers) {
        if (t.at > now) continue;
        if (!pick || t.at < pick.at || (t.at === pick.at && id < pickId)) {
          pick = t;
          pickId = id;
        }
      }
      if (!pick) return;
      if (pick.repeat) pick.at += pick.repeat;
      else timers.delete(pickId);
      try {
        pick.fn(...pick.args);
      } catch (err) {
        console.error("export timer", err);
      }
    }
  }

  function flushRAF(ts) {
    const list = rafs.splice(0);
    for (const fn of list) {
      try {
        fn(ts);
      } catch (err) {
        console.error("export raf", err);
      }
    }
  }

  const fakeNow = () => now;
  try {
    Object.defineProperty(performance, "now", { value: fakeNow, configurable: true });
  } catch (err) {
    try {
      performance.now = fakeNow;
    } catch (err2) {
      console.error("export clock: performance.now 不可替换", err2);
    }
  }
  Date.now = () => 1700000000000 + now;

  window.__clock = {
    get now() {
      return now;
    },
    set: setNow,
    run: runTimers,
    flush: flushRAF,
    // 演出期间不推进比赛时钟；离开时丢掉演出私有的定时器和 RAF。
    checkpoint: () => ({ timers: new Set(timers.keys()), rafCount: rafs.length }),
    discardSince(mark) {
      for (const id of timers.keys()) if (!mark.timers.has(id)) timers.delete(id);
      rafs.splice(mark.rafCount);
    },
  };
  window.__realRAF = realRAF;
  window.__realSetTimeout = realSetTimeout;
  window.__realClearTimeout = realClearTimeout;
})();
