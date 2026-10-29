// 原型机近战的装甲与战术扫描 HUD；只读快照，不参与索敌或碰撞结算。
window.lookFX = window.lookFX || {};

(function () {
  const A = window.arena;
  const plate = '<i class="melee-pod upper"></i><i class="melee-pod lower"></i>' +
    '<div class="melee-hull"><i class="melee-spine"></i><i class="melee-optic"></i>' +
    '<i class="melee-vent"></i><i class="melee-bolt upper"></i><i class="melee-bolt lower"></i></div>' +
    '<i class="melee-exhaust"></i>';
  const ticks = Array.from({ length: 72 }, (_, i) => {
    const major = i % 6 === 0;
    return `<path class="${major ? 'major' : 'minor'}" d="M200 ${major ? -4 : 0}v${major ? 12 : 4}" transform="rotate(${i * 5} 200 200)"/>`;
  }).join('');
  const radar = '<div class="melee-scan-pulses"><i></i><i></i><i></i></div>' +
    '<svg viewBox="0 0 400 400" aria-hidden="true">' +
    '<circle class="melee-range-line" cx="200" cy="200" r="200"/>' +
    '<circle class="melee-range-inset" cx="200" cy="200" r="189"/>' +
    `<g class="melee-range-ticks">${ticks}</g>` +
    '<g class="melee-range-gates"><path d="M157 9h-14l-12 5M243 9h14l12 5M157 391h-14l-12-5M243 391h14l12-5"/>' +
    '<path d="M9 157v-14l5-12M9 243v14l5 12M391 157v-14l-5-12M391 243v14l-5 12"/></g>' +
    '<g class="melee-heading"><path d="M359 193l13 7-13 7M351 196l8 4-8 4"/></g></svg>' +
    '<span class="melee-radar-state">SCAN</span><span class="melee-radar-range"></span>';

  function heading(u) {
    const moving = Math.hypot(u.vx || 0, u.vy || 0) > 1e-6;
    const dx = (moving ? u.vx : u.faceX) || 0;
    const dy = (moving ? u.vy : u.faceY) || 0;
    return Math.atan2(-dy, dx);
  }

  function targetFor(u, ctx) {
    if (typeof seekTarget !== 'function') return null;
    const r2 = u.vision * u.vision;
    const units = (ctx.units || []).filter(o =>
      o.ownerId === u.id || (o.x - u.x) ** 2 + (o.y - u.y) ** 2 <= r2);
    const walls = typeof state !== 'undefined' ? state.walls || [] : [];
    return seekTarget(u, units, walls);
  }

  function position(el, x, y, size, ctx) {
    const [sx, sy] = A.screenPos(x, y, ctx.scale, ctx.cx, ctx.cy);
    el.style.left = `${sx}px`;
    el.style.top = `${sy}px`;
    el.style.width = `${size}px`;
    el.style.height = `${size}px`;
    ctx.seenGuides.add(el.id);
  }

  window.lookFX.melee = {
    unmount(el) {
      el.querySelector(':scope > .melee-chassis')?.remove();
      el.classList.remove('melee-boosting');
      const ring = document.getElementById(`ring-${el.id.replace(/^u-/, '')}`);
      if (ring?.classList.contains('melee-radar')) ring.remove();
    },
    tick(el, u) {
      let chassis = el.querySelector(':scope > .melee-chassis');
      if (!chassis) {
        chassis = document.createElement('div');
        chassis.className = 'melee-chassis';
        chassis.setAttribute('aria-hidden', 'true');
        chassis.innerHTML = plate;
        el.appendChild(chassis);
      }
      chassis.style.transform = `rotate(${heading(u)}rad)`;
      const boost = Math.min(1, Math.max(0, (Math.hypot(u.vx || 0, u.vy || 0) - 195) / 140));
      el.style.setProperty('--melee-boost', boost.toFixed(3));
      el.classList.toggle('melee-boosting', boost > 0.6);
    },
    guide(u, ctx) {
      if (!(u.vision > 0)) return;
      const target = targetFor(u, ctx);
      // 复用既有视野圈 ID，避免原生圈与私有 HUD 重复，也支持已存在的圈升级外观。
      const el = ctx.ensureGuide(`ring-${u.id}`, 'melee-radar');
      el.classList.add('guide', 'melee-radar');
      if (!el.dataset.built) {
        el.innerHTML = radar;
        el.dataset.built = '1';
      }
      position(el, u.x, u.y, u.vision * ctx.scale * 2, ctx);
      el.querySelectorAll('.melee-scan-pulses i').forEach((pulse, i) => {
        const phase = (((ctx.time || 0) / 2.4 + i / 3) % 1 + 1) % 1;
        pulse.style.transform = `scale(${(0.08 + phase * 0.92).toFixed(4)})`;
        pulse.style.opacity = (Math.sin(phase * Math.PI) * 0.32).toFixed(4);
      });
      el.style.setProperty('--melee-heading', `${heading(u)}rad`);
      el.classList.toggle('acquired', !!target);
      el.querySelector('.melee-radar-state').textContent = target ? 'LOCK' : 'SCAN';
      el.querySelector('.melee-radar-range').textContent = `R ${Math.round(u.vision)}`;
      if (target) {
        const aim = ctx.ensureGuide(`melee-target-${u.id}`, 'melee-target');
        if (!aim.dataset.built) {
          aim.innerHTML = '<i></i><b></b>';
          aim.dataset.built = '1';
        }
        position(aim, target.x, target.y, (target.radius + 10) * ctx.scale * 2, ctx);
        aim.style.setProperty('--melee-lock-angle', `${(ctx.time || 0) * 28}deg`);
      }
    },
  };
})();
