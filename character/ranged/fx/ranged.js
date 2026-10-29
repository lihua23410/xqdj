// 远程原型机的炮台皮肤；瞄准只读快照，开火演出只响应真实 shot FX。
window.lookFX = window.lookFX || {};

(function () {
  const A = window.arena;
  const body = '<div class="ranged-base"><i class="ranged-tread upper"></i><i class="ranged-tread lower"></i>' +
    '<i class="ranged-deck"></i></div>' +
    '<div class="ranged-turret"><i class="ranged-cradle"></i>' +
    '<i class="ranged-optic"></i><i class="ranged-vent"></i>' +
    '<div class="ranged-cannon"><i class="ranged-barrel"></i><i class="ranged-muzzle"></i>' +
    '<i class="ranged-muzzle-fire"></i></div></div>';
  const collar = '<svg viewBox="0 0 100 100" aria-hidden="true">' +
    '<circle class="ranged-load-ring" cx="50" cy="50" r="45" pathLength="100"/></svg>';

  function heading(u) {
    const moving = Math.hypot(u.vx || 0, u.vy || 0) > 1e-6;
    return Math.atan2(-((moving ? u.vy : u.faceY) || 0), (moving ? u.vx : u.faceX) || 0);
  }

  function targetFor(u, ctx) {
    if (!(u.vision > 0) || typeof seekTarget !== 'function') return null;
    const r2 = u.vision * u.vision;
    const units = (ctx.units || []).filter(o =>
      o.ownerId === u.id || (o.x - u.x) ** 2 + (o.y - u.y) ** 2 <= r2);
    const walls = typeof state !== 'undefined' ? state.walls || [] : [];
    return seekTarget(u, units, walls);
  }

  function position(el, u, size, ctx) {
    const [x, y] = A.screenPos(u.x, u.y, ctx.scale, ctx.cx, ctx.cy);
    Object.assign(el.style, { left: `${x}px`, top: `${y}px`, width: `${size}px`, height: `${size}px` });
    ctx.seenGuides.add(el.id);
  }

  window.lookFX.ranged = {
    unmount(el) {
      el.querySelector(':scope > .ranged-machine')?.remove();
      el.classList.remove('ranged-locked', 'ranged-firing');
      document.getElementById(`ranged-control-${el.id.replace(/^u-/, '')}`)?.remove();
      document.getElementById(`ranged-target-${el.id.replace(/^u-/, '')}`)?.remove();
    },
    guide(u, ctx) {
      if (!(u.radius > 0)) return;
      // 射击历史挂在本机的私有导引节点上，随单位清理，不跨对局保留全局 ID 缓存。
      const el = ctx.ensureGuide(`ranged-control-${u.id}`, 'ranged-control');
      if (!el._ranged) {
        el.innerHTML = collar;
        el._ranged = {};
      }
      const data = el._ranged;
      const time = ctx.time || 0;
      if (time < data.time) delete data.shot;
      data.time = time;
      for (const fx of ctx.effects || []) {
        if (fx.name === 'shot' && fx.kind === u.kind && fx.unitId === u.id) {
          data.shot = { at: time, angle: Math.atan2(-(fx.vy || 0), fx.vx || 0) };
        }
      }
      data.target = targetFor(u, ctx);
      const load = data.shot ? Math.min(1, Math.max(0, (time - data.shot.at) / 1.6)) : 1;
      el.querySelector('.ranged-load-ring').style.strokeDashoffset = `${100 * (1 - load)}`;
      el.classList.toggle('reloading', load < 1);
      el.classList.toggle('acquired', !!data.target);
      position(el, u, (u.radius + 6) * ctx.scale * 2, ctx);
      if (data.target) {
        const aim = ctx.ensureGuide(`ranged-target-${u.id}`, 'ranged-target');
        if (!aim.firstChild) aim.innerHTML = '<i class="ranged-aim-corners"></i>';
        position(aim, data.target, (data.target.radius + 8) * ctx.scale * 2, ctx);
      }
    },
    tick(el, u, ctx) {
      let machine = el.querySelector(':scope > .ranged-machine');
      if (!machine) {
        machine = document.createElement('div');
        machine.className = 'ranged-machine';
        machine.setAttribute('aria-hidden', 'true');
        machine.innerHTML = body;
        el.appendChild(machine);
      }
      const data = document.getElementById(`ranged-control-${u.id}`)?._ranged;
      const shot = data?.shot;
      const target = data ? data.target : targetFor(u, ctx);
      const age = shot ? Math.max(0, (ctx.time || 0) - shot.at) : Infinity;
      let angle = target ? Math.atan2(-(target.y - u.y), target.x - u.x) : shot?.angle ?? heading(u);
      // 后退的底盘与炮塔分离；短暂保持真实出膛方向，避免炮口焰被换目标带走。
      if (age < 0.16) angle = shot.angle;
      machine.querySelector('.ranged-base').style.transform = `rotate(${heading(u)}rad)`;
      machine.querySelector('.ranged-turret').style.transform = `rotate(${angle}rad)`;
      const recoil = age < 0.045 ? 0.35 + age / 0.045 * 0.65 : Math.max(0, 1 - (age - 0.045) / 0.3) ** 2;
      el.style.setProperty('--ranged-recoil', recoil.toFixed(4));
      el.style.setProperty('--ranged-flash', Math.max(0, 1 - age / 0.13).toFixed(4));
      el.style.setProperty('--ranged-heat', Math.max(0, 1 - age / 0.8).toFixed(4));
      el.classList.toggle('ranged-locked', !!target);
      el.classList.toggle('ranged-firing', age < 0.13);
    },
  };
})();
