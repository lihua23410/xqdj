// 钉与锤：一次性特效登记（丑时参り）

const NAIL_SVG = `<svg viewBox="0 0 88 12" xmlns="http://www.w3.org/2000/svg">
  <rect x="1" y="2" width="9" height="8" fill="#9a9a9a" stroke="#2a2a2a" stroke-width="0.9"/>
  <rect x="2.2" y="3.2" width="6.6" height="2.2" fill="#e8e8e8" opacity="0.7"/>
  <path d="M9 4.6 H68 V7.4 H9 Z" fill="#b8b8b8" stroke="#2a2a2a" stroke-width="0.8"/>
  <path d="M12 5.15 H66" stroke="#f0f0f0" stroke-width="0.7" opacity="0.7"/>
  <polygon points="68,4.4 68,7.6 87,6" fill="#d0d0d0" stroke="#2a2a2a" stroke-width="0.8"/>
</svg>`;

const DOLL_SVG = `<svg viewBox="0 0 64 88" xmlns="http://www.w3.org/2000/svg">
  <path d="M24 54 L17 84" stroke="#7a5828" stroke-width="5.2" stroke-linecap="round"/>
  <path d="M40 54 L47 84" stroke="#8a6430" stroke-width="5.2" stroke-linecap="round"/>
  <ellipse cx="32" cy="46" rx="13" ry="17" fill="#c4a060" stroke="#5a3a18" stroke-width="1.2"/>
  <path d="M20 42 L6 34" stroke="#b08a48" stroke-width="4.4" stroke-linecap="round"/>
  <path d="M44 42 L58 34" stroke="#b08a48" stroke-width="4.4" stroke-linecap="round"/>
  <ellipse cx="32" cy="20" rx="10.5" ry="11.5" fill="#d4b878" stroke="#5a3a18" stroke-width="1.2"/>
  <path d="M27 18 L29 21" stroke="#1a1008" stroke-width="1.5" stroke-linecap="round"/>
  <path d="M37 18 L35 21" stroke="#1a1008" stroke-width="1.5" stroke-linecap="round"/>
  <rect x="27" y="36" width="10" height="15" fill="#f4ead4" stroke="#1a1208" stroke-width="0.8"/>
</svg>`;

function shotAng(fx) {
  const dx = Number(fx && fx.vx);
  const dy = Number(fx && fx.vy);
  const x = Number.isFinite(dx) ? dx : 0;
  const y = Number.isFinite(dy) ? dy : 0;
  if (Math.hypot(x, y) < 1e-6) return 0;
  return Math.atan2(-y, x);
}

arena.registerShot("hammer", (fx, ctx) => {
  const { x, y, kind } = ctx;
  const ball = document.getElementById(`u-${fx.unitId}`);
  const spin = ball && ball.querySelector(":scope > .hammer-art > .hammer-spin");
  if (spin) {
    spin.classList.remove("is-striking");
    void spin.offsetWidth;
    spin.classList.add("is-striking");
    spin.addEventListener("animationend", () => spin.classList.remove("is-striking"), { once: true });
  }
  arena.spawnFx("fx-shock", x, y, kind);
  arena.spawnFx("fx-ring", x, y, kind);
  arena.burst(x, y, kind, 10);
});

arena.registerShot("nail-spawn", (fx, ctx) => {
  const { x, y, kind } = ctx;
  const el = arena.spawnFx("fx-nail-spawn", x, y, kind, { "--ang": `${shotAng(fx)}rad` });
  if (el) el.innerHTML = NAIL_SVG;
  arena.spawnFx("fx-flash", x, y, kind);
});

arena.registerShot("nail-hit", (fx, ctx) => {
  const { x, y, kind } = ctx;
  const el = arena.spawnFx("fx-nail-hit", x, y, kind, { "--ang": `${shotAng(fx)}rad` });
  if (el) el.innerHTML = NAIL_SVG;
  arena.spawnFx("fx-shock", x, y, kind);
  arena.burst(x, y, kind, 10);
});

arena.registerShot("doll-spawn", (fx, ctx) => {
  const { x, y, kind } = ctx;
  const el = arena.spawnFx("fx-doll-spawn", x, y, kind);
  if (el) el.innerHTML = DOLL_SVG;
  arena.spawnFx("fx-flash", x, y, kind);
  arena.spawnFx("fx-ring", x, y, kind);
});
