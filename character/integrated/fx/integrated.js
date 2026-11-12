// 整合弹从分段外壳剥离。出弹方向取自 shot，追踪状态取自 track。
window.lookFX = window.lookFX || {};

(function () {
  const nearRadius = 185;
  const fleeRadius = 100;
  const cruise = 148;
  const trackSeconds = 5;
  const peelSeconds = 0.13;
  const plates = Array.from({ length: 8 }, (_, i) =>
    `<span class="integrated-segment" style="--segment-angle:${i * 45}deg"><i class="integrated-plate"></i></span>`).join('');
  const body = '<div class="integrated-body" aria-hidden="true"><i class="integrated-drive"></i>' +
    '<i class="integrated-core"></i><div class="integrated-shell"><i class="integrated-exposed"></i>' +
    plates + '</div></div>';
  function ticks(count, length) {
    return Array.from({ length: count }, (_, i) => {
      const angle = i * 2 * Math.PI / count;
      const x = Math.sin(angle);
      const y = -Math.cos(angle);
      return `M${(200 + x * 200).toFixed(2)} ${(200 + y * 200).toFixed(2)}l${(-x * length).toFixed(2)} ${(-y * length).toFixed(2)}`;
    }).join('');
  }
  const minorTicks = [ticks(60, 4), ticks(60, 8)];
  const majorTicks = [ticks(12, 9), ticks(12, 16)];
  const radar = '<svg viewBox="0 0 400 400" aria-hidden="true">' +
    '<circle class="integrated-range-line" cx="200" cy="200" r="200"/>' +
    '<circle class="integrated-range-inset" cx="200" cy="200" r="188"/>' +
    `<path class="integrated-ticks minor" d="${minorTicks[0]}"/><path class="integrated-ticks major" d="${majorTicks[0]}"/>` +
    '<path class="integrated-range-gates" d="M163 8h-15l-11 4M237 8h15l11 4M163 392h-15l-11-4M237 392h15l11-4"/>' +
    '<g class="integrated-heading"><path d="M365 194l10 6-10 6"/></g></svg>' +
    '<span class="integrated-range-label"></span>';
  const collar = '<svg viewBox="0 0 100 100" aria-hidden="true">' +
    '<circle class="integrated-load-ring" cx="50" cy="50" r="46" pathLength="100"/></svg>';

  function heading(u) {
    const moving = Math.hypot(u.vx || 0, u.vy || 0) > 1e-6;
    return Math.atan2(-((moving ? u.vy : u.faceY) || 0), (moving ? u.vx : u.faceX) || 0);
  }

  // 每帧从 track FX 重建，不把上一场或已消失主人的追踪留在全局表里。
  function tracks(ctx) {
    if (ctx._integratedTracks) return ctx._integratedTracks;
    const result = new Map();
    for (const fx of ctx.effects || []) {
      if (fx.name !== 'track' || !fx.unitId) continue;
      const target = (ctx.units || []).find(u => u.id === fx.vx && u.hp > 0);
      const owner = (ctx.units || []).find(u => u.id === fx.unitId && u.kind === fx.kind);
      if (owner && target && fx.amount > 0 && fx.vy > 0) result.set(fx.unitId, { target, remain: fx.vy });
      else result.delete(fx.unitId);
    }
    ctx._integratedTracks = result;
    return result;
  }

  function position(el, u, size, ctx) {
    Object.assign(el.style, {
      left: `${ctx.cx + u.x * ctx.scale}px`, top: `${ctx.cy - u.y * ctx.scale}px`,
      width: `${size}px`, height: `${size}px`,
    });
    ctx.seenGuides.add(el.id);
  }

  function placeRing(u, ctx, radius, inner, locked) {
    const el = ctx.ensureGuide(`integrated-${inner ? 'flee' : 'near'}-${u.id}`, `integrated-radar ${inner ? 'inner' : 'outer'}`);
    if (!el._integratedRing) {
      el.innerHTML = radar;
      el._integratedRing = {};
    }
    const data = el._integratedRing;
    const time = ctx.time || 0;
    // 与后端相同：两圈只看敌方战斗机或活随从的圆心距离。
    if (time < data.time) {
      delete data.inside;
      delete data.flashAt;
    }
    const inside = new Map();
    let engaged = false;
    for (const o of ctx.units || []) {
      if (o.slot === u.slot || o.nonsolid || !(o.hp > 0) || (o.role !== 'fighter' && !o.mortal)) continue;
      const close = Math.hypot(o.x - u.x, o.y - u.y) <= radius;
      inside.set(o.id, close);
      if (close) {
        engaged = true;
        if (data.inside?.get(o.id) === false) data.flashAt = time;
      }
    }
    if (engaged !== data.engaged) {
      el.querySelector('.integrated-ticks.minor').setAttribute('d', minorTicks[Number(engaged)]);
      el.querySelector('.integrated-ticks.major').setAttribute('d', majorTicks[Number(engaged)]);
    }
    data.engaged = engaged;
    data.inside = inside;
    data.time = time;
    const flash = data.flashAt == null ? 0 : Math.max(0, 1 - (time - data.flashAt) / 0.32);
    el.style.setProperty('--integrated-cross', flash.toFixed(3));
    el.style.setProperty('--integrated-minor-angle', `${time * (inner ? -360 / 5.6 : 360 / 7.2)}deg`);
    el.style.setProperty('--integrated-major-angle', `${time * (inner ? 360 / 11 : -360 / 14)}deg`);
    el.style.setProperty('--integrated-heading', `${heading(u)}rad`);
    el.classList.toggle('alert', engaged);
    const label = el.querySelector('.integrated-range-label');
    const status = inner ? 'EVADE' : locked ? 'TRACK' : 'SCAN';
    const text = `${status} / ${radius}`;
    if (label.textContent !== text) label.textContent = text;
    position(el, u, radius * ctx.scale * 2, ctx);
  }

  function placeTrack(u, ctx, lock) {
    if (!lock) return;
    const { target, remain } = lock;
    const el = ctx.ensureGuide(`integrated-mark-${u.id}`, 'integrated-mark');
    if (!el.firstChild) el.innerHTML = '<i class="integrated-track-arc"></i>';
    // 缺口保持朝上，弧从橙色尾端收短。
    el.style.setProperty('--arc', `${270 * Math.min(1, remain / trackSeconds)}deg`);
    position(el, target, (Math.max(target.radius || 18, 8) + 18) * ctx.scale * 2, ctx);
  }

  function findShot(owner, ctx) {
    return (ctx.effects || []).find(fx => fx.name === 'shot' && fx.kind === owner.kind && fx.unitId === owner.id);
  }

  function releaseFromShell(u, ctx) {
    const owner = (ctx.units || []).find(o => o.id === u.ownerId);
    const fx = owner && findShot(owner, ctx);
    if (!fx) return null;
    const angle = Math.atan2(-(fx.vy || 0), fx.vx || 0);
    // 与外壳上装甲片的中心 86.25% 对齐。
    const rim = owner.radius * 0.725;
    const x = (fx.x || 0) + Math.cos(angle) * rim;
    const y = (fx.y || 0) - Math.sin(angle) * rim;
    return { x: x - u.x, y: y - u.y };
  }

  window.lookFX.integrated = {
    guide(u, ctx) {
      if (u.role !== 'fighter') return;
      const time = ctx.time || 0;
      const lock = tracks(ctx).get(u.id);
      const el = ctx.ensureGuide(`integrated-control-${u.id}`, 'integrated-control');
      if (!el._integrated) {
        el.innerHTML = collar;
        el._integrated = {};
      }
      const data = el._integrated;
      if (time < data.time) delete data.shot;
      data.time = time;
      data.lock = lock;
      const fx = findShot(u, ctx);
      if (fx && data.shot?.at !== time) {
        data.shot = { at: time, angle: Math.atan2(-(fx.vy || 0), fx.vx || 0) };
      }
      const load = data.shot ? Math.min(1, Math.max(0, (time - data.shot.at) / 0.5)) : 1;
      el.querySelector('.integrated-load-ring').style.strokeDashoffset = `${100 * (1 - load)}`;
      el.classList.toggle('reloading', load < 1);
      position(el, u, (u.radius + 6) * ctx.scale * 2, ctx);
      placeRing(u, ctx, nearRadius, false, !!lock);
      placeRing(u, ctx, fleeRadius, true, !!lock);
      placeTrack(u, ctx, lock);
    },
    tick(el, u, ctx) {
      if (!el.querySelector(':scope > .integrated-body')) el.insertAdjacentHTML('beforeend', body);
      const data = document.getElementById(`integrated-control-${u.id}`)?._integrated;
      const shot = data?.shot;
      const age = shot ? Math.max(0, (ctx.time || 0) - shot.at) : Infinity;
      const plate = Math.min(1, Math.max(0, (age - peelSeconds) / (0.5 - peelSeconds)));
      const boost = Math.min(1, Math.max(0, (Math.hypot(u.vx || 0, u.vy || 0) - cruise) / 300));
      el.style.setProperty('--integrated-heading', `${heading(u)}rad`);
      el.style.setProperty('--integrated-shell-angle', `${shot?.angle ?? heading(u)}rad`);
      el.style.setProperty('--integrated-plate', plate.toFixed(3));
      el.style.setProperty('--integrated-boost', boost.toFixed(3));
      el.classList.toggle('integrated-boosting', boost > 0.1);
      el.classList.toggle('integrated-tracking', !!data?.lock);
    },
    unmount(el) {
      el.querySelector(':scope > .integrated-body')?.remove();
      el.classList.remove('integrated-boosting', 'integrated-tracking');
      const id = el.id.replace(/^u-/, '');
      for (const part of ['control', 'near', 'flee', 'mark']) document.getElementById(`integrated-${part}-${id}`)?.remove();
    },
  };

  window.lookFX['integrated-shot'] = {
    guide(u, ctx) {
      // 出弹和轨迹归属于导引节点，单位离场后由通用 seenGuides 清理。
      const el = ctx.ensureGuide(`integrated-flight-${u.id}`, 'integrated-flight');
      const time = ctx.time || 0;
      if (!el._integratedFlight || time < el._integratedFlight.time) {
        el._integratedFlight = { points: [], startedAt: time, release: releaseFromShell(u, ctx) };
      }
      const data = el._integratedFlight;
      const age = time - data.startedAt;
      data.peel = data.release ? Math.min(1, age / peelSeconds) : 1;
      data.roll = (u.id % 2 ? 1 : -1) * age * 5 * data.peel;
      data.offsetX = (data.release?.x || 0) * (1 - data.peel) ** 2;
      data.offsetY = (data.release?.y || 0) * (1 - data.peel) ** 2;
      ctx.seenGuides.add(el.id);
      if (!tracks(ctx).has(u.ownerId) || data.peel < 1) {
        el.firstChild?.remove();
        data.points.length = 0;
        data.time = time;
        return;
      }
      if (!el.firstChild) {
        el.innerHTML = '<svg aria-hidden="true"><polyline class="tail"/><polyline class="middle"/><polyline class="head"/></svg>';
      }
      // 储存世界坐标；缩放、导出画布改变时不会拉出一条错位长线。
      const angle = heading(u);
      const back = u.radius || 6;
      const point = { x: u.x - Math.cos(angle) * back, y: u.y + Math.sin(angle) * back };
      const last = data.points[data.points.length - 1];
      if (time !== data.time && (!last || Math.hypot(point.x - last.x, point.y - last.y) >= 2.5)) data.points.push(point);
      if (data.points.length > 18) data.points.splice(0, data.points.length - 18);
      data.time = time;
      const points = [...data.points, point];
      const stride = Math.ceil((points.length - 1) / 3);
      el.querySelectorAll('polyline').forEach((line, i) => {
        line.setAttribute('points', points.slice(i * stride, (i + 1) * stride + 1)
          .map(p => `${(ctx.cx + p.x * ctx.scale).toFixed(2)},${(ctx.cy - p.y * ctx.scale).toFixed(2)}`).join(' '));
      });
    },
    tick(el, u, ctx) {
      let shape = el.querySelector(':scope > .integrated-shot-shape');
      if (!shape) {
        shape = document.createElement('div');
        shape.className = 'integrated-shot-shape';
        shape.setAttribute('aria-hidden', 'true');
        shape.innerHTML = '<i class="integrated-shot-shell"></i>';
        el.appendChild(shape);
      }
      const speed = Math.hypot(u.vx || 0, u.vy || 0);
      const scale = ctx.scale || 1;
      const flight = document.getElementById(`integrated-flight-${u.id}`)?._integratedFlight;
      if (speed > 1e-6) el.style.setProperty('--shot-ang', `${heading(u)}rad`);
      el.style.setProperty('--shot-r', `${(u.radius || 6) * scale}px`);
      el.style.setProperty('--shot-peel', flight?.peel ?? 1);
      el.style.setProperty('--shot-roll', `${flight?.roll || 0}rad`);
      el.style.setProperty('--shot-dx', `${(flight?.offsetX || 0) * scale}px`);
      el.style.setProperty('--shot-dy', `${-(flight?.offsetY || 0) * scale}px`);
    },
    unmount(el) {
      el.querySelector(':scope > .integrated-shot-shape')?.remove();
      document.getElementById(`integrated-flight-${el.id.replace(/^u-/, '')}`)?.remove();
    },
  };

  // 演出由皮肤按模拟时间绘制，消费事件以免退化成无动画的临时 DOM。
  arena.registerShot('track', () => {});
  arena.registerShot('shot', () => {});
})();
