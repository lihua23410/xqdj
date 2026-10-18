// 反重力哈基米的 FX：
//   hajimi-fire 出弹小闪（倾斜场已删，那个箭头钩子一起删掉了）
arena.registerShot("hajimi-fire", (fx, ctx) => {
  arena.spawnFx("fx-hajimi-fire", ctx.x, ctx.y, ctx.kind);
});
