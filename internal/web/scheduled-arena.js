import { syncBattle } from "./scheduled-battle.js";

const query = new URLSearchParams(location.search);
const tournamentID = query.get("tournament");
const battleID = query.get("battle");
if (/^[a-f0-9]{32}$/.test(tournamentID) && /^[a-f0-9]{32}$/.test(battleID)) {
  document.querySelector("#scheduled-arena").hidden = false;
  document.querySelector("#back-to-tournament").href = "/tournament.html?id=" + tournamentID;
  const status = document.querySelector("#scheduled-arena-status");
  const check = async () => {
    try {
      const battle = await syncBattle(tournamentID, battleID);
      switch (battle.status) {
        case "finished":
          status.textContent = "对局已结束，返回赛程查看赛果";
          return;
        case "aborted":
          status.textContent = "对局已中止，返回赛程可重新开战";
          return;
        case "missing":
          status.textContent = "对局记录已不可用，请返回赛程";
          return;
        case "running":
          status.textContent = battle.field + " · 对局结束后自动录入";
          break;
      }
    } catch {
      status.textContent = "暂时无法保存赛果，正在重试";
    }
    setTimeout(check, 900);
  };
  check();
}
