// 镜像演出只读快照；历史随本机导引节点清理，时间使用比赛时间。
window.lookFX = window.lookFX || {};
(function () {
  const shell = '<div class="doppel-shell"><i class="doppel-core"><b></b></i>' +
    '<i class="doppel-refraction first"></i><i class="doppel-refraction second"></i></div>';
  const burst = '<svg viewBox="0 0 128 128" aria-hidden="true">' +
    '<circle class="doppel-burst-halo" cx="64" cy="64" r="38"/>' +
    '<circle class="doppel-burst-ring" cx="64" cy="64" r="47"/>' +
    '<g class="doppel-burst-shards"><path d="M18 38l15-9-5 15zM96 84l15 7-13 9zM44 15l11 9-14 2zM78 104l10-2-4 14zM12 77l15-5-7 14zM103 30l11 15-14-5z"/></g>' +
    '<path class="doppel-burst-slit" d="M44 82l14-27-5-10 26-22-10 32 6 9-17 30"/></svg>';
  window.lookFX.doppel = {
    guide(u, ctx) {
      const el = ctx.ensureGuide(`doppel-state-${u.id}`, 'doppel-state');
      const time = ctx.time || 0;
      if (!el._doppel || time < el._doppel.time) {
        el._doppel = { born: time, swap: -Infinity };
        el.innerHTML = burst;
      }
      const data = el._doppel;
      data.time = time;
      for (const fx of ctx.effects || []) {
        if (fx.name === 'swap' && fx.kind === '分身者' && fx.unitId === u.id) {
          data.swap = time;
          data.x = fx.x; data.y = fx.y;
        }
      }
      const age = time - data.swap;
      const active = age >= 0 && age < 0.42;
      el.style.display = active ? 'block' : 'none';
      if (active) {
        const phase = age / 0.42;
        const [x, y] = arena.screenPos(data.x, data.y, ctx.scale, ctx.cx, ctx.cy);
        const size = u.radius * 5.4 * ctx.scale;
        Object.assign(el.style, { left: `${x}px`, top: `${y}px`, width: `${size}px`, height: `${size}px`,
          opacity: `${Math.min(1, (1 - phase) * 1.5)}` });
        el.style.setProperty('--doppel-burst-spread', (0.78 + phase * 0.55).toFixed(4));
        el.style.setProperty('--doppel-burst-close', (1.1 - phase * 0.5).toFixed(4));
      }
      ctx.seenGuides.add(el.id);
    },
    tick(el, u, ctx = {}) {
      let art = el.querySelector(':scope > .doppel-shell');
      if (!art) {
        el.insertAdjacentHTML('beforeend', shell);
        art = el.querySelector(':scope > .doppel-shell');
      }
      const data = document.getElementById(`doppel-state-${u.id}`)?._doppel;
      const time = ctx.time || 0;
      const birth = data && u.role === 'clone' ? Math.max(0, 1 - (time - data.born) / 0.28) : 0;
      const swap = data ? Math.max(0, 1 - Math.max(0, time - data.swap - 0.06) / 0.3) : 0;
      art.style.setProperty('--doppel-split', (birth * 0.8 + swap).toFixed(4));
      art.style.setProperty('--doppel-birth', birth.toFixed(4));
      el.classList.toggle('doppel-copy', u.role === 'clone');
      el.classList.toggle('doppel-swapping', swap > 0);
    },
    unmount(el) {
      el.querySelector(':scope > .doppel-shell')?.remove();
      el.classList.remove('doppel-copy', 'doppel-swapping');
      document.getElementById(`doppel-state-${el.id.replace(/^u-/, '')}`)?.remove();
    },
  };
  window.lookFX['doppel-arc'] = {
    tick(el, u, ctx = {}) {
      let flow = el.querySelector(':scope > .doppel-orbit');
      if (!flow) {
        el.insertAdjacentHTML('beforeend', '<svg class="doppel-orbit" viewBox="0 0 100 100" aria-hidden="true">' +
          '<circle cx="50" cy="50" r="48"/></svg>');
        flow = el.querySelector(':scope > .doppel-orbit');
      }
      flow.style.transform = `rotate(${(ctx.time || 0) * 70}deg)`;
    },
    unmount(el) { el.querySelector(':scope > .doppel-orbit')?.remove(); },
  };
})();
