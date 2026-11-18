const storagePrefix = "xqdj:tournament:v2:";

function match(left, right) {
  return { left, right, result: "" };
}

export function defaultRoundCount(size) {
  return Math.min(size - 1, Math.max(3, Math.ceil(Math.log2(size)) - 1));
}

export function newBattleID() {
  return Array.from(crypto.getRandomValues(new Uint8Array(16)), byte =>
    byte.toString(16).padStart(2, "0")
  ).join("");
}

export function createTournament(groups, bye, roundCount) {
  const tournament = {
    id: newBattleID(), createdAt: Date.now(), groups, unassigned: bye, roundCount,
    rounds: [], playoff: null,
  };
  tournament.rounds.push(nextSwissRound(standings(tournament)));
  return tournament;
}

export function saveTournament(tournament) {
  localStorage.setItem(storagePrefix + tournament.id, JSON.stringify(tournament));
}

export function loadTournament(id) {
  return JSON.parse(localStorage.getItem(storagePrefix + id));
}

export function loadTournaments() {
  const tournaments = [];
  for (const key of Object.keys(localStorage)) {
    if (!key.startsWith(storagePrefix)) continue;
    const id = key.slice(storagePrefix.length);
    const data = localStorage.getItem(key);
    try {
      const tournament = JSON.parse(data);
      if (tournament?.id === id && tournament.rounds?.length && Array.isArray(tournament.groups)) {
        tournaments.push(tournament);
      }
    } catch {
      // A damaged entry should not hide other tournaments.
    }
  }
  return tournaments.sort((a, b) => b.createdAt - a.createdAt);
}

export function standings(tournament) {
  const rows = tournament.groups.map((_, seed) => ({
    id: String(seed + 1), seed, wins: 0, draws: 0, losses: 0, byes: 0, points: 0, sos: 0, opponents: [],
  }));
  const byID = new Map(rows.map(row => [row.id, row]));
  for (const { matches, bye } of tournament.rounds) {
    if (bye !== null) {
      const row = byID.get(bye);
      row.byes++;
      row.points++;
    }
    for (const { left, right, result } of matches) {
      if (!result) continue;
      const a = byID.get(left);
      const b = byID.get(right);
      a.opponents.push(right);
      b.opponents.push(left);
      if (result === "draw") {
        a.draws++;
        b.draws++;
        a.points += 0.5;
        b.points += 0.5;
      } else {
        const [winner, loser] = result === "left" ? [a, b] : [b, a];
        winner.wins++;
        winner.points++;
        loser.losses++;
      }
    }
  }
  for (const row of rows) {
    row.sos = row.opponents.reduce((sum, id) => sum + byID.get(id).points, 0);
  }
  return rows.sort((a, b) => b.points - a.points || b.sos - a.sos || b.wins - a.wins || a.seed - b.seed);
}

export function nextSwissRound(board) {
  const order = [...board];
  let bye = null;
  if (order.length % 2) {
    const fewestByes = Math.min(...order.map(row => row.byes));
    const index = order.findLastIndex(row => row.byes === fewestByes);
    bye = order.splice(index, 1)[0].id;
  }
  const matches = [];
  const hasPlayed = (a, b) => a.opponents.includes(b.id);
  while (order.length) {
    const a = order.shift();
    const index = order.findIndex(b => !hasPlayed(a, b));
    if (index >= 0) {
      matches.push([a, order.splice(index, 1)[0]]);
      continue;
    }
    // If the remaining opponents are repeats, try exchanging partners
    // with a pair already made before scheduling a rematch.
    const b = order.shift();
    const repair = matches.findLastIndex(([c, d]) =>
      (!hasPlayed(a, c) && !hasPlayed(b, d)) || (!hasPlayed(a, d) && !hasPlayed(b, c))
    );
    if (repair === -1) {
      matches.push([a, b]);
      continue;
    }
    const [c, d] = matches[repair];
    if (!hasPlayed(a, d) && !hasPlayed(b, c)) {
      matches[repair] = [c, b];
      matches.push([a, d]);
    } else {
      matches[repair] = [c, a];
      matches.push([b, d]);
    }
  }
  return { matches: matches.map(([a, b]) => match(a.id, b.id)), bye };
}

export function pending(round) {
  return round.matches.filter(game => !game.result).length;
}

export function winner(game) {
  if (game.result === "left") return game.left;
  if (game.result === "right") return game.right;
  if (game.result === "bye") return game.left ?? game.right;
  return null;
}

export function seedPlayoff(board) {
  const qualified = board.slice(0, 16).map(row => row.id);
  let seeds = [1, 2];
  while (seeds.length < qualified.length) {
    const complement = 2 * seeds.length + 1;
    seeds = seeds.flatMap(seed => [seed, complement - seed]);
  }
  const rounds = [[]];
  for (let i = 0; i < seeds.length; i += 2) {
    const game = match(qualified[seeds[i] - 1] ?? null, qualified[seeds[i + 1] - 1] ?? null);
    if (!game.left || !game.right) game.result = "bye";
    rounds[0].push(game);
  }
  return syncPlayoff(rounds);
}

export function syncPlayoff(rounds) {
  const next = [rounds[0]];
  while (next.at(-1).length > 1) {
    const previous = next.at(-1);
    const saved = rounds[next.length] ?? [];
    const games = [];
    for (let i = 0; i < previous.length; i += 2) {
      const game = match(winner(previous[i]), winner(previous[i + 1]));
      const old = saved[i / 2];
      games.push(old && old.left === game.left && old.right === game.right ? old : game);
    }
    next.push(games);
  }
  return next;
}
