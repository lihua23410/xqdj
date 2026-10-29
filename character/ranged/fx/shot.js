(function () {
  const src = (document.currentScript && document.currentScript.dataset.pack) || "";
  const shotVoice = arena.voice(new Audio(src + "/fx/shot.wav"), {
    mode: arena.SOUND_MODE.IMMUNE,
    priority: 0,
  });
  shotVoice.audio.preload = "auto";

  arena.registerShot("shot", (fx, ctx) => {
    const visualOnly = !!(ctx && ctx.visualOnly);
    if (!visualOnly) {
      shotVoice.stop();
      shotVoice.reset();
      shotVoice.play();
    }
    // 炮口焰、后坐和散热由 ranged 皮肤读取同一 shot FX，按比赛时间推进。
  });
})();
