// Private samples use the shared arbiter; packet traffic shares one rate limit.
(function () {
  const A = window.arena;
  const base = document.currentScript?.dataset.pack || '/ball/中转站';
  const definitions = {
    neighbor: {mode:A.SOUND_MODE.IMMUNE, volume:.6, voices:2, gap:.35},
    forward: {mode:A.SOUND_MODE.NORMAL, priority:210, volume:.38, voices:2, gap:.16},
    ack: {mode:A.SOUND_MODE.NORMAL, priority:210, volume:.38, voices:2, gap:.16},
    hit: {mode:A.SOUND_MODE.NORMAL, priority:205, volume:.55, voices:2, gap:.16}
  };
  const sounds = {};
  for (const [name,opts] of Object.entries(definitions)) {
    sounds[name] = {opts,next:0,last:-Infinity,pool:Array.from({length:opts.voices},() => {
      const audio = new Audio(`${base}/fx/${name}.wav`);
      audio.preload='auto';
      return A.voice(audio,opts);
    })};
  }
  let lastTraffic = -Infinity;
  let visualTime = -Infinity;
  const flashes = new Map();
  function flashAllowed(fx, time) {
    const now = Number.isFinite(time) ? time : 0;
    if (now < visualTime) flashes.clear();
    visualTime = now;
    for (const [id,at] of flashes) if (now-at > 2) flashes.delete(id);
    const last = flashes.get(fx.unitId) ?? -Infinity;
    if (now-last+1e-9 < .12) return false;
    flashes.set(fx.unitId,now);
    return true;
  }
  function play(name, ctx) {
    const s=sounds[name], now=Number.isFinite(ctx.time)?ctx.time:performance.now()/1000;
    if (now>=s.last && now-s.last+1e-9<s.opts.gap) return;
    if (name==='forward' || name==='ack') {
      if (now>=lastTraffic && now-lastTraffic+1e-9<.16) return;
      lastTraffic=now;
    }
    s.last=now;
    const voice=s.pool[s.next++%s.pool.length];
    voice.stop(); voice.reset(); voice.play();
  }
  for (const [event,sound,shape] of [
    ['join','neighbor','neighbor'],['forward','forward','tx'],['arrive','ack','rx'],['hit','hit','hit']
  ]) {
    A.registerShot(`relay-${event}`, (fx,ctx) => {
      if (event!=='hit') window.lookFX?.relay?.notify(event,fx,ctx.time);
      play(sound,ctx);
      // Port state and real routes always update; merge dense transient rings.
      if (event!=='join' && !flashAllowed(fx,ctx.time)) return;
      const scale=ctx.scale||1;
      const el=A.spawnFx(`fx-relay-${shape}`,ctx.x,ctx.y,ctx.kind,{
        '--relay-size':`${(event==='join'?30:event==='hit'?24:22)*scale}px`
      });
      if (el && event==='join') el.innerHTML='<span>HELLO</span>';
      if (el && event==='hit') el.innerHTML='<i></i><i></i><i></i><i></i>';
    });
  }
})();
