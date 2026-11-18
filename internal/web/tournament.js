import {
  loadTournament, loadTournaments, newBattleID, nextSwissRound, pending, saveTournament,
  seedPlayoff, standings, syncPlayoff, winner,
} from "./tournament-core.js";
import { battleLink, gameAt, startBattle, syncBattle } from "./scheduled-battle.js";

const selectTournament = document.querySelector("#tournament");
const selectRound = document.querySelector("#round");
const summary = document.querySelector("#summary");
const message = document.querySelector("#message");
const competition = document.querySelector("#competition");
const advance = document.querySelector("#advance");
const playoff = document.querySelector("#playoff");
let tournament;
let roundIndex;
let launching = false;
let polling = false;

function element(tag, text, className) {
  const node = document.createElement(tag);
  if (text != null) node.textContent = text;
  if (className) node.className = className;
  return node;
}

function option(text, value) {
  const node = element("option", text);
  node.value = value;
  return node;
}

function tournamentLabel(t) {
  const date = new Date(t.createdAt).toLocaleString("zh-CN");
  return `${date} · ${t.groups.length} 组`;
}

function groupLabel(id) {
  return `第 ${id} 组`;
}

function members(id) {
  return tournament.groups[Number(id) - 1];
}

function groupName(id) {
  return `${groupLabel(id)}（${members(id).join("、")}）`;
}

function load() {
  let all;
  try {
    all = loadTournaments();
  } catch {
    summary.textContent = "无法读取浏览器保存的赛程。";
    competition.hidden = true;
    return;
  }
  document.querySelector("#tournament-picker").hidden = all.length < 2;
  selectTournament.replaceChildren(...all.map(t => option(tournamentLabel(t), t.id)));
  const id = new URL(location.href).searchParams.get("id");
  const next = id ? all.find(t => t.id === id) : all[0];
  if (!next) {
    summary.textContent = id ? "这个赛程不在当前浏览器中。" : "还没有赛程。回到随机分组页，用一轮抽签开赛。";
    competition.hidden = true;
    tournament = undefined;
    if (all.length) {
      document.querySelector("#tournament-picker").hidden = false;
      selectTournament.prepend(option("选择已保存的赛程", ""));
      selectTournament.value = "";
    }
    return;
  }
  if (next.id !== tournament?.id) roundIndex = next.rounds.length - 1;
  roundIndex = Math.min(roundIndex, next.rounds.length - 1);
  tournament = next;
  selectTournament.value = tournament.id;
  render();
}

function commit(next, view = roundIndex) {
  try {
    const saved = loadTournament(tournament.id);
    if (JSON.stringify(saved) !== JSON.stringify(tournament)) {
      load();
      message.textContent = "这个赛程已在另一页面更新，请核对后重试。";
      return false;
    }
    saveTournament(next);
  } catch {
    message.textContent = "赛程保存失败，请检查浏览器站点存储。";
    return false;
  }
  tournament = next;
  roundIndex = view;
  message.textContent = "";
  render();
  return true;
}

function updateSwiss(index, value) {
  if (tournament.active) return false;
  const next = structuredClone(tournament);
  const match = next.rounds[roundIndex].matches[index];
  if (match.result === value) return false;
  if (next.rounds.length > roundIndex + 1 || next.playoff) {
    if (!confirm("修改这场结果将清除后续瑞士轮和淘汰赛。确定修改？")) return false;
    next.rounds.length = roundIndex + 1;
    next.playoff = null;
  }
  match.result = value;
  return commit(next);
}

function updatePlayoff(stage, index, value) {
  if (tournament.active) return false;
  const next = structuredClone(tournament);
  next.playoff[stage][index].result = value;
  next.playoff = syncPlayoff(next.playoff);
  const losesResults = tournament.playoff.slice(stage + 1).some((games, i) =>
    games.some((game, j) => game.result && !next.playoff[stage + i + 1][j].result)
  );
  if (losesResults && !confirm("修改这场结果将清除受影响的后续赛果。确定修改？")) return false;
  return commit(next);
}

async function launch(game, stage, index) {
  if (launching || tournament.active || game.result) return;
  const id = newBattleID();
  const tournamentID = tournament.id;
  const next = structuredClone(tournament);
  next.active = { id, stage, round: roundIndex, index, requestedAt: Date.now() };
  const sides = [members(game.left), members(game.right)];
  launching = true;
  if (!commit(next)) {
    launching = false;
    return;
  }
  message.textContent = "正在准备对局…";
  try {
    const response = await startBattle(id, sides);
    if (response.ok) {
      location.href = battleLink(tournamentID, id);
      return;
    }
    const reason = (await response.text()).trim();
    const saved = loadTournament(tournamentID);
    if (saved?.active?.id === id) {
      delete saved.active;
      saveTournament(saved);
    }
    load();
    message.textContent = reason || "对局未能启动。";
  } catch {
    const view = document.querySelector("[data-active-battle]");
    if (view) view.hidden = false;
    message.textContent = "连接未完成，正在核对对局状态。";
  } finally {
    launching = false;
  }
}

function groupSide(id, fallback, won) {
  const side = element("div", null, won ? "group-side won" : "group-side");
  side.append(element("strong", id ? groupLabel(id) : fallback));
  if (id) side.append(element("span", members(id).join("、"), "members"));
  return side;
}

function showBattleState(game, swiss) {
  const battle = game.battle;
  if (!battle) return null;
  if (battle.status === "missing") return "对局记录已失效，可重新开战";
  if (battle.status === "aborted") return battle.field + " · 已中止，可重新开战";
  if (!swiss && battle.result === "draw" && !game.result) {
    return battle.field + " · 平局，请重赛或手动录入胜者";
  }
  const result = game.result === battle.result ? "自动录入" : "已手动更正";
  return battle.field + " · " + result;
}

function matchCard(game, index, stage) {
  const card = element("div", null, "match");
  const won = winner(game);
  const fallback = game.result === "bye" ? "轮空" : "待定";
  card.append(groupSide(game.left, fallback, !!won && won === game.left));
  card.append(element("span", "对", "versus"));
  card.append(groupSide(game.right, fallback, !!won && won === game.right));
  if (game.result === "bye") {
    card.append(element("p", groupLabel(won) + "晋级"));
    return card;
  }
  if (!game.left || !game.right) return card;
  const swiss = stage === null;
  const result = element("select");
  result.id = swiss ? `swiss-${roundIndex}-${index}` : `playoff-${stage}-${index}`;
  result.setAttribute("aria-label", `${groupLabel(game.left)}对${groupLabel(game.right)}的赛果`);
  result.append(option("待录入", ""), option(groupLabel(game.left) + "胜", "left"));
  if (swiss) result.append(option("平局", "draw"));
  result.append(option(groupLabel(game.right) + "胜", "right"));
  result.disabled = Boolean(tournament.active);
  result.value = game.result;
  result.addEventListener("change", () => {
    const changed = swiss ? updateSwiss(index, result.value) : updatePlayoff(stage, index, result.value);
    if (changed) document.getElementById(result.id)?.focus();
    else result.value = game.result;
  });
  const actions = element("div", null, "match-actions");
  actions.append(result);
  if (tournament.active && gameAt(tournament, tournament.active) === game) {
    const view = element("a", "进入战斗", "battle-link");
    view.href = battleLink(tournament.id, tournament.active.id);
    view.dataset.activeBattle = "";
    if (launching) view.hidden = true;
    actions.append(view);
  } else if (!game.result) {
    const start = element("button", "开战");
    start.type = "button";
    start.disabled = Boolean(tournament.active);
    start.addEventListener("click", () => launch(game, stage, index));
    actions.append(start);
  }
  card.append(actions);
  const details = showBattleState(game, swiss);
  if (details) card.append(element("p", details, "battle-details"));
  return card;
}

function renderStandings(board) {
  document.querySelector("#standings").replaceChildren(...board.map((row, i) => {
    const entry = element("tr");
    if (tournament.playoff && i < 16) entry.className = "qualified";
    const values = [i + 1, groupName(row.id), row.points, row.sos, row.wins, row.draws, row.losses, row.byes];
    entry.append(...values.map(value => element("td", value)));
    return entry;
  }));
}

function renderPlayoff() {
  playoff.hidden = !tournament.playoff;
  if (!tournament.playoff) return;
  const rounds = tournament.playoff;
  const champion = winner(rounds.at(-1)[0]);
  document.querySelector("#champion").textContent = champion
    ? "冠军 " + groupName(champion)
    : "按瑞士轮名次排签，淘汰赛须分出胜负。";
  const labels = { 8: "十六强", 4: "八强", 2: "半决赛", 1: "决赛" };
  const tree = element("div", null, "bracket-tree");
  tree.style.setProperty("--slots", rounds[0].length);
  tree.append(...rounds.map((games, stage) => {
    const round = element("section", null, "tree-column");
    const cards = element("ol", null, "tree-games");
    cards.append(...games.map((game, i) => {
      const node = element("li", null, "tree-node");
      const card = matchCard(game, i, stage);
      card.dataset.round = labels[games.length];
      card.setAttribute("aria-label", `${labels[games.length]}第 ${i + 1} 场`);
      node.append(card);
      return node;
    }));
    round.append(element("h3", labels[games.length]), cards);
    return round;
  }));
  document.querySelector("#bracket").replaceChildren(tree);
}

function render() {
  competition.hidden = false;
  const qualifying = Math.min(tournament.groups.length, 16);
  const unassigned = tournament.unassigned ? ` · 候补 ${tournament.unassigned}` : "";
  summary.textContent = `${tournamentLabel(tournament)} · 瑞士轮 ${tournament.roundCount} 轮 · 前 ${qualifying} 组晋级${unassigned}`;
  const current = tournament.rounds[roundIndex];
  selectRound.replaceChildren(...tournament.rounds.map((_, i) => option(`第 ${i + 1} 轮`, i)));
  selectRound.value = roundIndex;
  const unfinished = pending(current);
  const bye = current.bye ? ` · 轮空 ${groupName(current.bye)}` : "";
  document.querySelector("#round-status").textContent = `${current.matches.length} 场 · ${unfinished} 场待录入${bye}`;
  document.querySelector("#swiss-matches").replaceChildren(...current.matches.map((game, i) => matchCard(game, i, null)));
  advance.hidden = roundIndex !== tournament.rounds.length - 1 || Boolean(tournament.playoff);
  advance.disabled = unfinished > 0 || Boolean(tournament.active);
  advance.textContent = tournament.rounds.length < tournament.roundCount
    ? "生成下一轮"
    : qualifying === 16 ? "生成十六强" : "生成淘汰赛";
  renderStandings(standings(tournament));
  renderPlayoff();
}

advance.addEventListener("click", () => {
  if (advance.hidden || advance.disabled) return;
  const next = structuredClone(tournament);
  const board = standings(next);
  if (next.rounds.length < next.roundCount) {
    next.rounds.push(nextSwissRound(board));
    commit(next, next.rounds.length - 1);
  } else {
    next.playoff = seedPlayoff(board);
    if (commit(next)) playoff.scrollIntoView({ block: "start" });
  }
});

selectRound.addEventListener("change", () => {
  roundIndex = Number(selectRound.value);
  render();
});

selectTournament.addEventListener("change", () => {
  if (!selectTournament.value) return;
  const url = new URL(location.href);
  url.searchParams.set("id", selectTournament.value);
  history.replaceState(null, "", url);
  message.textContent = "";
  load();
});

window.addEventListener("storage", load);
window.addEventListener("pageshow", event => {
  if (event.persisted) load();
});
setInterval(async () => {
  if (!tournament?.active || launching || polling) return;
  const { id, active } = tournament;
  polling = true;
  try {
    const battle = await syncBattle(id, active.id);
    if (tournament?.id !== id || tournament.active?.id !== active.id) return;
    if (battle.status === "running") {
      message.textContent = battle.field + " · 正在对战";
    } else if (battle.status !== "starting") {
      load();
      message.textContent = battle.status === "finished" ? "对局已结算" : "对局未结算，可重新开战。";
    }
  } catch {
    if (tournament?.id === id) message.textContent = "暂时无法读取战斗状态，正在重试。";
  } finally {
    polling = false;
  }
}, 900);
load();
