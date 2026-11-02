"""Continue after the serial screen; simulation stages never overlap."""
import json
import os
import subprocess
import time

import analyze

ROOT = analyze.ROOT


def run_stage(name, n, jobs, extra=()):
    marker = ROOT / (name + "-run.json")
    if marker.exists():
        assert json.loads(marker.read_text())["exit_code"] == 0
        return
    args = ["/tmp/xqdj-winrate-audit-target", "-n", str(n), "-timeout", "90", "-j", str(jobs), "-o", str(ROOT / (name + ".md")), "-audit", str(ROOT / (name + ".jsonl")), *extra]
    print("START", name, flush=True)
    (ROOT / "stage.json").write_text(json.dumps(dict(stage=name, args=args, started=time.time()), indent=2))
    start = time.monotonic()
    child = subprocess.Popen(args, cwd=ROOT.parents[1])
    _, status, usage = os.wait4(child.pid, 0)
    child.returncode = os.waitstatus_to_exitcode(status)
    result = dict(args=args, wall_seconds=time.monotonic() - start, peak_rss_kb=usage.ru_maxrss, exit_code=child.returncode)
    marker.write_text(json.dumps(result, indent=2))
    assert child.returncode == 0
    print("DONE", name, json.dumps(result), flush=True)


if __name__ == "__main__":
    while not (ROOT / "serial-10-run.json").exists():
        time.sleep(1)
    assert json.loads((ROOT / "serial-10-run.json").read_text())["exit_code"] == 0
    serial = analyze.load_games("serial-10.jsonl")
    screen = analyze.screen(serial)
    assert screen["rounds"] == 10 and screen["games"] == 14310
    (ROOT / "screen.json").write_text(json.dumps(screen, ensure_ascii=False, indent=2))
    selected = screen["pairs"][:12]
    selected_keys = {(p["a"], p["b"]) for p in selected}
    controls = [p for p in screen["pairs"] if (p["a"], p["b"]) not in selected_keys and .4 <= p["original"][1] / p["original"][0] <= .6]
    selected += sorted(controls, key=lambda p: p["max_delta_pp"])[:4]
    assert len(selected) == 16
    targets = [{"A": p["a"], "B": p["b"]} for p in selected]
    (ROOT / "targets.json").write_text(json.dumps(targets, ensure_ascii=False, indent=2))
    print("TARGETS", json.dumps(targets, ensure_ascii=False), flush=True)
    run_stage("parallel-10", 10, 32)
    extra = ["-pairs", str(ROOT / "targets.json"), "-offset", "100"]
    run_stage("serial-targets", 200, 1, extra)
    run_stage("parallel-targets", 200, 32, extra)
    for name, sname in (("parallel-10", "serial-10"), ("parallel-targets", "serial-targets")):
        result = analyze.paired(analyze.load_games(sname + ".jsonl"), analyze.load_games(name + ".jsonl"))
        (ROOT / (name + "-paired.json")).write_text(json.dumps(result, ensure_ascii=False, indent=2))
    (ROOT / "stage.json").write_text(json.dumps(dict(stage="complete", finished=time.time()), indent=2))
    print("ALL VALIDATION STAGES COMPLETE", flush=True)
