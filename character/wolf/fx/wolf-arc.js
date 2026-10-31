// 所有爪痕/獠牙裁在快照给出的真实扇环内，朝向仍由公共渲染器处理。
(function () {
  const hooks = window.lookFX = window.lookFX || {};
  const point = (r, a) => `${(r * Math.cos(a)).toFixed(4)} ${(r * Math.sin(a)).toFixed(4)}`;
  function sector(inner, outer, half) {
    const large = half > Math.PI / 2 ? 1 : 0;
    return `M${point(outer, -half)}A${outer} ${outer} 0 ${large} 1 ${point(outer, half)}L${point(inner, half)}A${inner} ${inner} 0 ${large} 0 ${point(inner, -half)}Z`;
  }
  function claw(r, width, a, b) {
    return `M${point(r, a)}A${r} ${r} 0 0 1 ${point(r, b)}L${point(r-width, b-.10)}A${r-width} ${r-width} 0 0 0 ${point(r-width, a+.20)}Z`;
  }
  function draw(art, u, bite) {
    const outer = u.radius;
    const inner = u.arcInner || 0;
    const half = u.arcSpan / 2;
    const thick = outer - inner;
    const clip = `wolf-arc-clip-${u.id}`;
    art.setAttribute("viewBox", `${-outer} ${-outer} ${outer*2} ${outer*2}`);
    art.innerHTML = `<defs><clipPath id="${clip}"><path d="${sector(inner,outer,half)}"/></clipPath></defs><g clip-path="url(#${clip})">
      <path class="wolf-arc-field" d="${sector(inner,outer,half)}"/>
      ${bite ? `<path class="wolf-jaw-rim" d="M${point(outer-.5,-half)}A${outer-.5} ${outer-.5} 0 0 1 ${point(outer-.5,half)}"/>
        <path class="wolf-gum" d="M${point(outer-1,-half)}A${outer-1} ${outer-1} 0 0 1 ${point(outer-1,half)}"/>
        ${[-.86,-.57,-.28,.28,.57,.86].map((n,i)=>{
          const a = n*half;
          const base = outer-1.3;
          const depth = i === 1 || i === 4 ? .93 : .69;
          return `<path class="wolf-arc-fang" d="M${point(base,a-.085)}L${point(inner+thick*(1-depth),a+.025)}L${point(base,a+.085)}Z"/>`;
        }).join("")}
        <path class="wolf-jaw-edge" d="M${point(inner+.45,-half)}A${inner+.45} ${inner+.45} 0 0 1 ${point(inner+.45,half)}"/>`
        : [0,1,2].map(i=>`<path class="wolf-claw wolf-claw-${i}" d="${claw(outer-.35-i*thick*.29,thick*.17,-half+.07+i*.10,half-.07-i*.11)}"/>`).join("")}
      <path class="wolf-arc-glint" d="M${point(outer-.3,-half)}A${outer-.3} ${outer-.3} 0 0 1 ${point(outer-.3,half)}"/>
    </g>`;
  }
  hooks["wolf-arc"] = {
    tick(el, u, ctx) {
      let art = el.querySelector(":scope > .wolf-arc-art");
      if (!art) {
        art = document.createElementNS("http://www.w3.org/2000/svg", "svg");
        art.classList.add("wolf-arc-art");
        art.setAttribute("aria-hidden", "true");
        el.appendChild(art);
      }
      const time = ctx.time || 0;
      if (el._wolfArcTime == null || time < el._wolfArcTime) el._wolfArcBorn = time;
      el._wolfArcTime = time;
      const bite = u.kind === "狼人撕咬弧";
      const key = `${u.radius}:${u.arcInner}:${u.arcSpan}:${bite}`;
      if (art._wolfGeometry !== key) {
        art._wolfGeometry = key;
        draw(art, u, bite);
      }
      const snap = Math.max(0, 1 - (time - el._wolfArcBorn) / .18);
      el.classList.toggle("wolf-bite-arc", bite);
      el.style.setProperty("--wolf-snap", snap.toFixed(4));
    },
    unmount(el) {
      el.querySelector(":scope > .wolf-arc-art")?.remove();
      el.classList.remove("wolf-bite-arc");
      el.style.removeProperty("--wolf-snap");
      delete el._wolfArcBorn;
      delete el._wolfArcTime;
    },
  };
})();
