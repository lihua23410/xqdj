"""Audit report counts, seeds and serial/parallel distributions. No game code changes."""
import collections
import json
import math
from pathlib import Path

from scipy.stats import binomtest, fisher_exact, spearmanr

ROOT = Path(__file__).resolve().parent
PRIME = 1099511628211
MASK = (1 << 64) - 1
INVERSE = pow(PRIME, -1, 1 << 64)


def original():
    text = (ROOT / "parallel-original.md").read_text()
    sections = dict(x.split("\n", 1) for x in text.split("\n## ")[1:])
    pairs = {}
    solo = {}
    for line in sections["单球胜率"].splitlines():
        c = [x.strip() for x in line.strip("|").split("|")]
        if len(c) == 7 and c[1].isdigit():
            solo[c[0]] = list(map(int, c[1:5]))
    for line in sections["一对一明细"].splitlines():
        c = [x.strip() for x in line.strip("|").split("|")]
        if len(c) == 8 and c[1].isdigit():
            p = tuple(c[0].split(" vs "))
            assert p not in pairs
            pairs[p] = list(map(int, c[1:6]))
    assert len(pairs) == len(solo) * (len(solo) - 1) // 2
    totals = collections.defaultdict(lambda: [0, 0, 0, 0])
    for (a, b), (n, wa, wb, d, to) in pairs.items():
        assert n == 100 and wa + wb + d == n and 0 <= to <= d
        for k, w, loss in ((a, wa, wb), (b, wb, wa)):
            totals[k] = [x + y for x, y in zip(totals[k], (n, w, loss, d))]
    assert dict(totals) == solo
    for title, with_draw in (("一对一胜率", False), ("一对一胜平率", True)):
        lines = [l for l in sections[title].splitlines() if l.startswith("|")]
        cols = [x.strip() for x in lines[0].strip("|").split("|")][1:]
        assert set(cols) == set(solo)
        for line in lines[2:]:
            cells = [x.strip() for x in line.strip("|").split("|")]
            a = cells[0]
            assert len(cells) == len(cols) + 1
            for b, value in zip(cols, cells[1:]):
                if a == b:
                    assert value == "—"
                    continue
                p = (a, b) if (a, b) in pairs else (b, a)
                n, wa, wb, d, _ = pairs[p]
                w = wa if a == p[0] else wb
                assert value == f"{100 * (w + (d if with_draw else 0)) / n:.1f}%"
    return pairs, solo


BASE, SOLO = original()
CANON = {frozenset(p): p for p in BASE}


def name_hash(left, right):
    h = 14695981039346656037
    for char in left + "\0" + right:
        h = ((h ^ ord(char)) * PRIME) & MASK
    return h


def load_games(name):
    path = ROOT / name
    games = {}
    if not path.exists():
        return games
    for line in path.read_text().splitlines():
        try:
            r = json.loads(line)
        except json.JSONDecodeError:
            continue  # Reading a running writer's last partial record.
        p = CANON[frozenset((r["left"], r["right"]))]
        round_no = (((r["seed"] * INVERSE) & MASK) ^ name_hash(r["left"], r["right"])) - 1
        assert 0 <= round_no < 10000
        assert (r["left"], r["right"]) == (p if round_no % 2 == 0 else p[::-1])
        key = (*p, round_no)
        assert key not in games, (name, key)
        assert r["winner"] in (*p, "平局", "")
        assert 1 <= r["ticks"] <= 5400
        r["outcome"] = 0 if r["winner"] == p[0] else 1 if r["winner"] == p[1] else 2
        games[key] = r
    return games


def aggregate(games):
    counts = collections.defaultdict(lambda: [0, 0, 0, 0, 0])
    for (a, b, _), r in games.items():
        c = counts[a, b]
        c[0] += 1
        c[1 + r["outcome"]] += 1
        c[4] += int(r["winner"] == "平局" and r["ticks"] >= 5400)
    return dict(counts)


def holm(rows, field="p"):
    previous = 0
    for i, row in enumerate(sorted(rows, key=lambda r: r[field])):
        previous = max(previous, min(1.0, row[field] * (len(rows) - i)))
        row["p_holm"] = previous


def screen(games):
    counts = aggregate(games)
    assert set(counts) == set(BASE)
    n = min(c[0] for c in counts.values())
    n -= n % 2
    games = {k: v for k, v in games.items() if k[2] < n}
    counts = aggregate(games)
    if not n:
        return None
    assert all(c[0] == n for c in counts.values())
    totals = collections.defaultdict(lambda: [0, 0, 0, 0])
    pair_rows = []
    tests = []
    for (a, b), c in counts.items():
        for kind, w, loss in ((a, c[1], c[2]), (b, c[2], c[1])):
            totals[kind] = [x + y for x, y in zip(totals[kind], (n, w, loss, c[3]))]
        old = BASE[a, b]
        delta = [(c[i] / n - old[i] / old[0]) * 100 for i in (1, 2, 3)]
        pair_rows.append(dict(a=a, b=b, original=old, serial=c, delta_pp=delta, max_delta_pp=max(map(abs, delta))))
        for i in (1, 2, 3):
            p = float(fisher_exact([[old[i], old[0] - old[i]], [c[i], n - c[i]]]).pvalue)
            tests.append(dict(a=a, b=b, outcome=i, p=p))
    holm(tests)
    fighters = []
    for kind, c in totals.items():
        old = SOLO[kind]
        fighters.append(dict(kind=kind, original=old, serial=c, delta_pp=100 * (c[1] / c[0] - old[1] / old[0])))
    return dict(rounds=n, games=len(games), fighters=sorted(fighters, key=lambda r: abs(r["delta_pp"]), reverse=True), pairs=sorted(pair_rows, key=lambda r: r["max_delta_pp"], reverse=True), pair_tests=tests, significant_pair_tests=sum(r["p_holm"] < .05 for r in tests), spearman=float(spearmanr([SOLO[k][1] / SOLO[k][0] for k in totals], [totals[k][1] / totals[k][0] for k in totals]).statistic), draws=sum(c[3] for c in counts.values()), timeouts=sum(c[4] for c in counts.values()))


def paired(serial, parallel):
    assert set(serial) == set(parallel), "paired runs have different seeds"
    ps = collections.defaultdict(lambda: [0, 0, 0])  # n, serial-only, parallel-only
    fs = collections.defaultdict(lambda: [0, 0, 0])
    changed = 0
    for key, s in serial.items():
        p = parallel[key]
        assert s["seed"] == p["seed"]
        changed += s["outcome"] != p["outcome"]
        for outcome in range(3):
            a, b = s["outcome"] == outcome, p["outcome"] == outcome
            row = ps[(*key[:2], outcome)]
            row[0] += 1
            row[1] += a and not b
            row[2] += b and not a
        for outcome, kind in enumerate(key[:2]):
            a, b = s["outcome"] == outcome, p["outcome"] == outcome
            row = fs[kind]
            row[0] += 1
            row[1] += a and not b
            row[2] += b and not a
    def result(label, counts):
        n, up, down = counts
        return dict(label=label, n=n, serial_only=up, parallel_only=down, delta_pp=100 * (up - down) / n, p=float(binomtest(up, up + down, .5).pvalue) if up + down else 1.)
    pairs = [result(k, v) for k, v in ps.items()]
    fighters = [result(k, v) for k, v in fs.items()]
    holm(pairs)
    holm(fighters)
    return dict(games=len(serial), changed_outcomes=changed, pair_tests=sorted(pairs, key=lambda r: r["p_holm"]), fighters=sorted(fighters, key=lambda r: abs(r["delta_pp"]), reverse=True), significant_pair_tests=sum(r["p_holm"] < .05 for r in pairs), significant_fighters=sum(r["p_holm"] < .05 for r in fighters))


if __name__ == "__main__":
    serial = load_games("serial-10.jsonl")
    data = screen(serial)
    if data:
        (ROOT / "screen.json").write_text(json.dumps(data, ensure_ascii=False, indent=2))
        print("screen", data["rounds"], "rounds", data["games"], "games; Spearman", round(data["spearman"], 4), "Holm significant endpoints", data["significant_pair_tests"])
        for r in data["fighters"][:10]:
            print(r["kind"], f'{r["original"][1]/r["original"][0]:.1%}', f'{r["serial"][1]/r["serial"][0]:.1%}', f'{r["delta_pp"]:+.1f} pp')
    for prefix in ("parallel-10", "parallel-targets"):
        sname = "serial-10.jsonl" if prefix == "parallel-10" else "serial-targets.jsonl"
        if (ROOT / (prefix + "-run.json")).exists() and (ROOT / sname).exists():
            s, p = load_games(sname), load_games(prefix + ".jsonl")
            comparison = paired(s, p)
            (ROOT / (prefix + "-paired.json")).write_text(json.dumps(comparison, ensure_ascii=False, indent=2))
            print(prefix, comparison["games"], "paired games; Holm significant fighters", comparison["significant_fighters"], "endpoints", comparison["significant_pair_tests"])
