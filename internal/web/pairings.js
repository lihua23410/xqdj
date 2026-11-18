import { createTournament, defaultRoundCount, saveTournament } from "./tournament-core.js";

(() => {
  const redraw = document.querySelector("#redraw");
  const status = document.querySelector("#status");
  const pairs = document.querySelector("#pairs");
  const bye = document.querySelector("#bye");
  const byeName = document.querySelector("#bye-name");
  const start = document.querySelector("#start-tournament");
  const roundCount = document.querySelector("#round-count");
  const tournamentError = document.querySelector("#tournament-error");
  const storageKey = "xqdj:last-pairing";
  let lastDraw;
  let currentDraw;

  function draw(roster) {
    const shuffled = [...roster];
    for (let end = shuffled.length - 1; end > 0; end--) {
      const pick = Math.floor(Math.random() * (end + 1));
      [shuffled[end], shuffled[pick]] = [shuffled[pick], shuffled[end]];
    }
    const leftover = shuffled.length % 2 ? shuffled.pop() : null;
    const groups = [];
    for (let i = 0; i < shuffled.length; i += 2) {
      groups.push(shuffled.slice(i, i + 2));
    }
    // A different display order alone does not count as a new grouping.
    const signature = JSON.stringify([
      groups.map(group => JSON.stringify([...group].sort())).sort(), leftover,
    ]);
    return { groups, leftover, signature };
  }

  function distinctDraw(roster) {
    let previous = lastDraw;
    try {
      previous = localStorage.getItem(storageKey) ?? previous;
    } catch {
      // Storage may be unavailable; still avoid repeats within this page.
    }
    let result;
    do {
      result = draw(roster);
    } while (roster.length >= 3 && result.signature === previous);
    lastDraw = result.signature;
    try {
      localStorage.setItem(storageKey, lastDraw);
    } catch {
      // The grouping can still be displayed.
    }
    return result;
  }

  function label(className, text) {
    const span = document.createElement("span");
    span.className = className;
    span.textContent = text;
    return span;
  }

  function show({ groups, leftover }) {
    pairs.replaceChildren(...groups.map(([first, second], i) => {
      const item = document.createElement("li");
      const members = document.createElement("div");
      members.className = "members";
      members.append(label("name", first), label("plus", "+"), label("name", second));
      item.append(label("group", `第 ${i + 1} 组`), members);
      return item;
    }));
    byeName.textContent = leftover ?? "";
    bye.hidden = leftover === null;
  }

  async function refresh() {
    if (redraw.disabled) return;
    redraw.disabled = true;
    status.removeAttribute("data-error");
    status.textContent = "正在读取战斗机名单…";
    currentDraw = null;
    start.hidden = true;
    tournamentError.textContent = "";
    pairs.replaceChildren();
    bye.hidden = true;
    try {
      const response = await fetch("/api/pairing-roster", { cache: "no-store" });
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      const roster = await response.json();
      const result = distinctDraw(roster);
      show(result);
      if (result.groups.length >= 2) {
        currentDraw = result;
        const max = result.groups.length - 1;
        if (Number(roundCount.max) !== max) roundCount.value = defaultRoundCount(result.groups.length);
        roundCount.max = max;
        start.hidden = false;
      }
      status.textContent = roster.length
        ? `${roster.length} 只战斗机 · ${result.groups.length} 组${result.leftover === null ? "" : " · 1 只未分组"}`
        : "当前名单没有可分组的战斗机。";
    } catch {
      status.setAttribute("data-error", "");
      status.textContent = "读取名单失败。请从项目根目录启动服务后重试。";
    } finally {
      redraw.disabled = false;
    }
  }

  start.addEventListener("submit", event => {
    event.preventDefault();
    if (!currentDraw) return;
    const { groups, leftover } = currentDraw;
    const tournament = createTournament(groups, leftover, roundCount.valueAsNumber);
    try {
      saveTournament(tournament);
    } catch {
      tournamentError.textContent = "浏览器无法保存赛程。请检查站点存储设置。";
      return;
    }
    location.href = "/tournament.html?id=" + tournament.id;
  });

  redraw.addEventListener("click", refresh);
  window.addEventListener("pageshow", event => {
    if (event.persisted) refresh();
  });
  refresh();
})();
