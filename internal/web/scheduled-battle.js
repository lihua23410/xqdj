import { loadTournament, saveTournament, syncPlayoff } from "./tournament-core.js";

export function gameAt(tournament, { stage, round, index }) {
  return stage === null
    ? tournament.rounds[round]?.matches[index]
    : tournament.playoff?.[stage]?.[index];
}

function endpoint(id) {
  return "/api/scheduled-battles/" + encodeURIComponent(id);
}

export function battleLink(tournamentID, battleID) {
  return "/team?" + new URLSearchParams({ tournament: tournamentID, battle: battleID });
}

export async function startBattle(id, sides) {
  return fetch(endpoint(id), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ sides }),
  });
}

export async function syncBattle(tournamentID, battleID) {
  const response = await fetch(endpoint(battleID), { cache: "no-store" });
  if (!response.ok && response.status !== 404) throw new Error("无法读取对局状态");
  const battle = response.ok ? await response.json() : { status: "missing" };
  if (battle.status === "running") return battle;

  const latest = loadTournament(tournamentID);
  const active = latest?.active;
  if (active?.id !== battleID) return battle;
  // Another tab may poll before the start request reaches the server.
  if (battle.status === "missing" && Date.now() - active.requestedAt < 5000) {
    return { status: "starting" };
  }
  const game = gameAt(latest, active);
  if (game) {
    game.battle = battle;
    if (battle.status === "finished" && !game.result) {
      if (active.stage === null || battle.result !== "draw") {
        game.result = battle.result;
        if (active.stage !== null) latest.playoff = syncPlayoff(latest.playoff);
      }
    }
  }
  delete latest.active;
  saveTournament(latest);
  return battle;
}
