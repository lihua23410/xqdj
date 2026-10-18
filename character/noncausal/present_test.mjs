import test from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import vm from "node:vm";
import path from "node:path";
import { fileURLToPath } from "node:url";

const dir = path.dirname(fileURLToPath(import.meta.url));

function load(file, extra) {
  const code = fs.readFileSync(path.join(dir, file), "utf8");
  const context = {
    console,
    performance: { now: () => extra.now || 0 },
    document: extra.document || { addEventListener() {}, querySelector() { return null; } },
    ...extra,
  };
  context.window = extra.window || context;
  context.globalThis = context;
  vm.createContext(context);
  vm.runInContext(code, context);
  return context;
}

const present = load("fx/noncausal.js", {});
const P = present.NoncausalPresent;

test("caption, opacity band, rewind cap", () => {
  assert.equal(P.CAPTION, "打的是对面那条死掉的世界线");
  assert.ok(P.GHOST_OPACITY >= 0.25 && P.GHOST_OPACITY <= 0.35);
  assert.equal(P.introDuration(10), 3);
  assert.equal(P.introDuration(1.2), 1.2);
});

test("spectrum fields are cadence and counts, not damage", () => {
  const st = P.createState();
  P.applyEffect(st, { name: "spectrum", amount: 0.62, vx: 3, vy: 8, slot: 0, unitId: 1 }, 0, 12);
  assert.equal(st.cadence, 0.62);
  assert.equal(st.binCount, 3);
  assert.equal(st.pulses, 8);
  assert.equal(st.bins.length, 3);
  P.applyEffect(st, { name: "spectrum-bin", amount: 7, vx: 4, vy: 0.62, slot: 1, unitId: 1 }, 0, 12);
  assert.equal(st.bins[1].amount, 7);
  assert.equal(st.bins[1].weight, 4);
  assert.equal(st.hot, -1);
});

test("doom-at starts T† and freezes the track", () => {
  const st = P.createState();
  P.sampleTrack(st, 1, 0, 0);
  P.sampleTrack(st, 2, 10, 0);
  P.applyEffect(st, { name: "doom-at", amount: 9.5, vx: 100, vy: 0.62, slot: 0, unitId: 1 }, 0, 20);
  assert.equal(st.doomAt, 9.5);
  assert.equal(st.ghostMax, 100);
  assert.equal(st.ghostHP, 100);
  assert.equal(st.tracking, false);
  assert.equal(st.frozenTrack.length, 2);
  assert.equal(P.tDaggerText(st, 22.5), "T† 7.0");
  assert.equal(P.tDaggerText(st, 40), "T† 0.0");
});

test("lock-shot highlights slot and does not read velocity as binIndex", () => {
  const st = P.createState();
  P.applyEffect(st, { name: "spectrum", amount: 0.5, vx: 5, vy: 4, unitId: 1 }, 0, 1);
  for (const [i, amount] of [2, 4, 7, 11, 18].entries()) {
    P.applyEffect(st, { name: "spectrum-bin", slot: i, amount, vx: 1, vy: 0.5, unitId: 1 }, 0, 1);
  }
  const fire = P.applyEffect(
    st,
    { name: "lock-shot", slot: 4, amount: 18, vx: 200, vy: -30, x: 1, y: 2, unitId: 1 },
    1000,
    3,
  );
  assert.equal(st.hot, 4);
  assert.equal(st.drawnAmount, 18);
  assert.equal(fire.flash, "spawn");
  assert.ok(fire.thick > P.thicknessForTier(0, 5));
  const merged = P.applyEffect(
    st,
    { name: "lock-shot", slot: 4, amount: 18, vx: 10, vy: 10, unitId: 1 },
    1040,
    3,
  );
  assert.equal(merged.flash, "merge");
  const small = P.applyEffect(
    st,
    { name: "lock-shot", slot: 0, amount: 2, vx: 999, vy: 0, unitId: 1 },
    1200,
    3,
  );
  assert.equal(small.flash, "none");
  assert.equal(st.hot, 0);
});

test("lock-block is a sever and ghost-hit is a tear with remaining HP", () => {
  P.reset();
  P.bindResidual(9, 1);
  const st = P.stateFor(1);
  P.applyEffect(st, { name: "doom-at", amount: 8, vx: 40, vy: 0.5, unitId: 1 }, 0, 1);
  const sever = P.note({ name: "lock-block", amount: 0, vx: 0, vy: 0, slot: 2, unitId: 8 });
  const tear = P.note({ name: "ghost-hit", amount: 7, vx: 33, vy: 8, slot: 2, unitId: 9 });
  assert.equal(sever.kind, "sever");
  assert.equal(tear.kind, "tear");
  assert.equal(st.ghostHP, 33);
  assert.equal(P.torn(9, 10), true);
  assert.equal(P.torn(9, 5000), false);
});

test("rewind scrubs backward and stops after 3s", () => {
  const track = [0, 1, 2, 3, 4].map((i) => ({ t: i, x: i * 10, y: 0 }));
  assert.equal(P.rewindFrame(track, 0, false).point.x, 40);
  assert.equal(P.rewindFrame(track, 3, false).point.x, 0);
  assert.equal(P.rewindFrame(track, 3.01, false).active, false);
  assert.equal(P.rewindFrame(track, 0.2, true).active, false);
});

test("shot registry uses the backend names", () => {
  const shots = {};
  const spawned = [];
  const arena = {
    registerShot(name, fn) {
      shots[name] = fn;
    },
    spawnFx(cls, x, y, kind) {
      const el = { className: cls, textContent: "", classList: { add() {} }, style: {}, offsetWidth: 1 };
      spawned.push({ cls, x, y, kind, el });
      return el;
    },
  };
  load("fx/shot.js", {
    window: present,
    arena,
    document: { addEventListener() {}, querySelector() { return null; } },
  });
  for (const name of [
    "spectrum",
    "spectrum-bin",
    "doom-at",
    "probe-hit",
    "lock-shot",
    "lock-armed",
    "lock-block",
    "ghost-hit",
    "ghost-down",
    "doom",
  ]) {
    assert.equal(typeof shots[name], "function", name);
  }
  shots.spectrum({ name: "spectrum", amount: 0.4, vx: 1, vy: 1, unitId: 3 });
  shots["lock-block"]({ name: "lock-block", amount: 0, vx: 0, vy: 1, slot: 0, unitId: 9 }, { x: 4, y: 5, kind: "锁影弹" });
  shots["ghost-hit"]({ name: "ghost-hit", amount: 6, vx: 10, vy: 9, slot: 0, unitId: 4 }, { x: 1, y: 1, kind: "残影" });
  assert.equal(spawned[0].cls, "nc-sever");
  assert.equal(spawned[0].el.textContent, "因果被截");
  assert.equal(spawned[1].cls, "nc-tear");
  assert.equal(spawned[1].el.textContent, "");
  assert.equal(P.stateFor(3).cadence, 0.4);
});
