// 反重力哈基米的 FX：出弹闪光、10 秒重力脉冲、弹体方向扭转标记。
arena.registerShot("hajimi-fire", (fx, ctx) => {
  arena.spawnFx("fx-hajimi-fire", ctx.x, ctx.y, ctx.kind);
});

arena.registerShot("gravity-field", (fx, ctx) => {
  const root = document.getElementById("fx");
  if (!root) return;
  const scale = ctx.scale || 1;
  const diameter = Math.max(80, (Number(fx.amount) || 420) * 2 * scale);
	  const duration = Math.max(0.2, Number(fx.vx) || 3);
	  arena.spawnFx("fx-hajimi-field", ctx.x, ctx.y, ctx.kind, {
	    "--r": `${diameter}px`,
	    "--duration": `${duration}s`,
	  }, root);
  arena.spawnFx("fx-hajimi-field-ring", ctx.x, ctx.y, ctx.kind, {
	    "--duration": `${duration}s`,
    "--r": `${diameter}px`,
    "--delay": "0.12s",
  }, root);
  arena.burst(ctx.x, ctx.y, ctx.kind, 28);
});

arena.registerShot("gravity-twist", (fx, ctx) => {
  const root = document.getElementById("fx");
  if (!root) return;
  const ang = Math.atan2(-(Number(fx.vy) || 0), Number(fx.vx) || 1);
  arena.spawnFx("fx-hajimi-twist", ctx.x, ctx.y, ctx.kind, { "--ang": `${ang}rad` }, root);
});

arena.registerShot("gravity-convert", (fx, ctx) => {
  const root = document.getElementById("fx");
  if (!root) return;
  const ang = Math.atan2(-(Number(fx.vy) || 0), Number(fx.vx) || 1);
  arena.spawnFx("fx-hajimi-twist", ctx.x, ctx.y, ctx.kind, { "--ang": `${ang}rad` }, root);
  arena.spawnFx("fx-flash", ctx.x, ctx.y, ctx.kind);
  arena.burst(ctx.x, ctx.y, ctx.kind, 10);
});

arena.registerShot("gravity-capture", (fx, ctx) => {
  const root = document.getElementById("fx");
  if (!root) return;
  const ang = Math.atan2(-(Number(fx.vy) || 0), Number(fx.vx) || 1);
  arena.spawnFx("fx-hajimi-twist", ctx.x, ctx.y, ctx.kind, { "--ang": `${ang}rad` }, root);
});

arena.registerShot("gravity-release", (fx, ctx) => {
  const root = document.getElementById("fx");
  if (!root) return;
  const ang = Math.atan2(-(Number(fx.vy) || 0), Number(fx.vx) || 1);
  arena.spawnFx("fx-hajimi-twist", ctx.x, ctx.y, ctx.kind, { "--ang": `${ang}rad` }, root);
  arena.spawnFx("fx-hajimi-fire", ctx.x, ctx.y, ctx.kind);
  arena.burst(ctx.x, ctx.y, ctx.kind, 8);
});
