// 非因果私有的探针倒带播放器。公共调度器不解释 death 和轨迹数据。
const acausalRewindAudio = ((document.currentScript && document.currentScript.dataset.pack) || "/ball/非因果") + "/fx/probe-hum.mp3";
arena.registerPresentation("rewind", {
  mount(ctx, film) {
    if (!film || !Array.isArray(film.frames) || !film.frames.length || !(film.death >= 0)) throw new Error("无效的非因果演出数据");
    ctx.root.classList.add("acausal-rewind");
    ctx.playAudio(acausalRewindAudio, { mode: arena.SOUND_MODE.FORCE, loop: true });
    return { film };
  },
  render(instance, elapsed, ctx) {
    const film=instance.film;
    const death=Number(film.death);
    const playDur=0.5+death/10;
    const layer=ctx.root;
    const {scale,cx,cy}=ctx;
    let t;
    if (elapsed < 0.5) t = Math.max(0, death - 0.5) + elapsed;
    else if (elapsed < playDur) t = Math.max(0, death - (elapsed - 0.5) * 10);
    else t = 0;
    let frame = film.frames[0];
    let best = Infinity;
    for (const f of film.frames) {
      const d = Math.abs(Number(f.t) - t);
      if (d < best) {
        best = d;
        frame = f;
      }
    }
    const bodies = frame.b || [];
    while (layer.childElementCount < bodies.length) {
      const el = document.createElement("div");
      el.className = "intro-ball";
      layer.appendChild(el);
    }
    while (layer.childElementCount > bodies.length) { ctx.unmountLookFX(layer.lastChild); layer.lastChild.remove(); }
    bodies.forEach((b, i) => {
      const el = layer.children[i];
      const r = (b.r || 18) * scale;
      const [sx, sy] = ctx.screenPos(b.x, b.y, scale, cx, cy);
      el.style.width = `${r * 2}px`;
      el.style.height = `${r * 2}px`;
      el.style.left = `${sx}px`;
      el.style.top = `${sy}px`;
      el.style.background = ctx.kindColor(b.k);
      el.style.boxShadow = "0 0 10px " + ctx.kindColor(b.k);
      ctx.paintKind(el, b.k);
      ctx.applyLookFX(el, b.k);
      ctx.runLookFX(el, { id: "presentation-" + ctx.id + "-" + i, kind: b.k, x: b.x, y: b.y, radius: b.r || 18, slot: b.s }, { ...ctx, now: elapsed * 1000, time: elapsed, units: [], fxRoot: layer });
    });
  },
  destroy(instance, ctx) {
    for (const el of ctx.root.children) ctx.unmountLookFX(el);
    if (window.gsap) window.gsap.killTweensOf(ctx.root.querySelectorAll("*"));
  },
});
