// 钉与锤：一次性特效登记（丑时参り）— GSAP 版
// 对比旧版：CSS keyframes + class toggle → GSAP timeline，参数全在 JS 里动态调
// 位置坑：spawnFx 的 .fx 用 CSS translate(-50%,-50%) 居中，GSAP 接管 transform 会覆盖它，
// 所以统一用 gsap.set(el, { xPercent: -50, yPercent: -50 }) 定位，其余交给 GSAP 分开管理。

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
    // 旧版：remove class → reflow → add class → animationend 清理，角度写死在 CSS keyframes 里
    // 新版：一个 timeline 说清，参数随便调；再演示 GSAP 独有的收招回弹（CSS 要加关键帧才做得出来）
    gsap.timeline({ overwrite: true })
      .fromTo(spin, { rotation: 0 }, { rotation: -360, duration: 0.34, ease: "power3.out" })
      .to(spin, { rotation: -348, duration: 0.09, ease: "power2.out" })
      .to(spin, { rotation: -360, duration: 0.07, ease: "power1.inOut" });
  }
  arena.spawnFx("fx-shock", x, y, kind);
  arena.spawnFx("fx-ring", x, y, kind);
  arena.burst(x, y, kind, 10);
});

arena.registerShot("nail-spawn", (fx, ctx) => {
  const { x, y, kind } = ctx;
  const el = arena.spawnFx("fx-nail-spawn", x, y, kind, { "--ang": `${shotAng(fx)}rad` });
  if (el) {
    el.innerHTML = NAIL_SVG;
    // 旧版 keyframes：scale 0.4 → 1.2 → 0.95 + 淡出，0.34s ease-out，写死在 CSS 里
    gsap.set(el, { xPercent: -50, yPercent: -50 });
    gsap.timeline({ onComplete: () => el.remove() })
      .fromTo(el, { scale: 0.4, opacity: 1 },
        { scale: 1.2, duration: 0.19, ease: "power2.out" })
      .to(el, { scale: 0.95, opacity: 0, duration: 0.15, ease: "power1.out" });
  }
  arena.spawnFx("fx-flash", x, y, kind);
});

arena.registerShot("nail-hit", (fx, ctx) => {
  const { x, y, kind } = ctx;
  const el = arena.spawnFx("fx-nail-hit", x, y, kind, { "--ang": `${shotAng(fx)}rad` });
  if (el) {
    el.innerHTML = NAIL_SVG;
    // 旧版 keyframes：整钉往里顶一下再钉穿（-10% → 8% → 16%），现在"顶进深度"是参数
    gsap.set(el, { xPercent: -50, yPercent: -50, x: "-10%" });
    gsap.timeline({ onComplete: () => el.remove() })
      .to(el, { x: "8%", duration: 0.2, ease: "power2.out" })
      .to(el, { x: "16%", opacity: 0, duration: 0.3, ease: "power1.out" });
  }
  arena.spawnFx("fx-shock", x, y, kind);
  arena.burst(x, y, kind, 10);
});

arena.registerShot("doll-spawn", (fx, ctx) => {
  const { x, y, kind } = ctx;
  const el = arena.spawnFx("fx-doll-spawn", x, y, kind);
  if (el) {
    el.innerHTML = DOLL_SVG;
    // 旧版 keyframes：缩放弹出 + 旋转修正 + 淡出三段，0.7s
    gsap.set(el, { xPercent: -50, yPercent: -50 });
    gsap.timeline({ onComplete: () => el.remove() })
      .fromTo(el, { scale: 0.2, rotation: -8, opacity: 0 },
        { scale: 1.15, rotation: 3, opacity: 0.95, duration: 0.28, ease: "power2.out" })
      .to(el, { scale: 1.45, rotation: 0, opacity: 0, duration: 0.42, ease: "power1.out" });
  }
  arena.spawnFx("fx-flash", x, y, kind);
  arena.spawnFx("fx-ring", x, y, kind);
});
