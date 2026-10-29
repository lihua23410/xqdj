// 开场演出调度。播放器只解释自己的 data，并按给定 elapsed 绘制。
// 禁止以真实时间推进剧情；动画、音频和外部资源通过 context 交给调度器回收。
(function () {
  const players = new Map();
  let active = null;
  const key = (base, type) => `${base}/${type}`;

  function register(base, type, player) {
    if (!base || !type || !player || !["mount", "render", "destroy"].every((m) => typeof player[m] === "function")) {
      throw new Error("演出播放器需要资源包、类型和 mount/render/destroy");
    }
    const id = key(base, type);
    if (players.has(id)) throw new Error(`演出播放器重复注册：${id}`);
    players.set(id, player);
  }

  function clear() {
    if (!active) return;
    const old = active;
    active = null;
    let error;
    try { old.player.destroy(old.instance, old.ctx); } catch (e) { error = e; }
    for (const dispose of old.disposers.reverse()) {
      try { dispose(); } catch (e) { error = error || e; }
    }
    old.root.remove();
    if (error) throw error;
  }

  function draw(list, elapsed, context) {
    const { createVoice, ...sharedContext } = context;
    let segment;
    let local = Math.max(0, elapsed);
    for (const item of list || []) {
      if (!(item.duration > 0) || !Number.isFinite(item.duration)) throw new Error("无效的演出时长");
      if (local < item.duration) { segment = item; break; }
      local -= item.duration;
    }
    if (!segment) { clear(); return false; }
    if (!active || active.id !== segment.id || active.preview !== !!context.preview) {
      clear();
      const player = players.get(key(`/ball/${segment.pack}`, segment.type));
      if (!player) throw new Error(`找不到演出播放器：${segment.pack}/${segment.type}`);
      const root = document.createElement("div");
      root.className = "presentation-layer";
      root.dataset.presentation = segment.type;
      context.host.appendChild(root);
      const disposers = [];
      const animations = [];
      const ctx = {
        ...sharedContext, root, id: segment.id, elapsed: local,
        duration: segment.duration, mode: context.mode,
        own(dispose) {
          if (typeof dispose !== "function") throw new Error("演出资源需要清理函数");
          disposers.push(dispose);
        },
        listen(target, event, fn, options) {
          target.addEventListener(event, fn, options);
          disposers.push(() => target.removeEventListener(event, fn, options));
        },
        animate(el, frames, options) {
          const animation = el.animate(frames, options);
          animation.pause();
          animations.push(animation);
          disposers.push(() => animation.cancel());
          return animation;
        },
        playAudio(src, options) {
          if (!segment.audio || context.preview) return null;
          const voice = context.createVoice(src, options);
          disposers.push(() => voice.dispose());
          return voice;
        },
      };
      active = { id: segment.id, preview: !!context.preview, player, root, ctx, disposers, animations, instance: null };
      try { active.instance = player.mount(ctx, segment.data); }
      catch (err) { try { clear(); } finally { throw err; } }
    }
    Object.assign(active.ctx, sharedContext, { elapsed: local });
    try {
      active.player.render(active.instance, local, active.ctx);
      for (const animation of active.animations) animation.currentTime = local * 1000;
    } catch (err) { try { clear(); } finally { throw err; } }
    return true;
  }

  window.presentationPlayer = { register, draw, clear };
})();
