(function () {
  window.lookFX = window.lookFX || {};
  window.engineHeat = window.engineHeat || {};

  // A continuous uneven flame front reads as fire rather than repeated particle icons.
  const flameOutline = Array.from({ length: 144 }, (_, i) => {
    const a = i / 144 * Math.PI * 2;
    const tongue = Math.pow(Math.max(0, Math.sin(a * 11 + 0.6)), 10);
    const radius = 95 + Math.sin(a * 23) * 1.3 + tongue * (6 + 5 * Math.sin(a * 3 + 1) ** 2);
    return `${i ? "L" : "M"}${(Math.sin(a) * radius).toFixed(2)} ${(-Math.cos(a) * radius).toFixed(2)}`;
  }).join("") + "Z" + Array.from({ length: 144 }, (_, i) => {
    const a = (143 - i) / 144 * Math.PI * 2;
    const radius = 91 + Math.sin(a * 17) * 0.6;
    return `${i ? "L" : "M"}${(Math.sin(a) * radius).toFixed(2)} ${(-Math.cos(a) * radius).toFixed(2)}`;
  }).join("") + "Z";

  function mount(el, id) {
    const machine = document.createElement("div");
    machine.className = "engine-machine";
    machine.setAttribute("aria-hidden", "true");
    machine.innerHTML = `<svg viewBox="0 0 100 100" focusable="false">
      <circle class="engine-case" cx="50" cy="50" r="47"/>
      <circle class="engine-case-seam" cx="50" cy="50" r="42"/>
      <circle class="engine-jacket" cx="50" cy="50" r="35"/>
      ${[0, 90, 180, 270].map((angle) => `<g transform="rotate(${angle} 50 50)">
        <rect class="engine-cylinder" x="37" y="12" width="26" height="28" rx="4"/>
        <path class="engine-fins" d="M36 17h28M35 23h30M36 29h28"/>
        <rect class="engine-bore" x="42" y="15" width="16" height="27" rx="2"/>
        <g class="engine-piston">
          <rect class="engine-rod" x="47.5" y="25" width="5" height="25" rx="2"/>
          <rect class="engine-piston-head" x="41" y="18" width="18" height="10" rx="2"/>
          <path class="engine-piston-groove" d="M42 21h16M42 24h16"/>
        </g>
      </g>`).join("")}
      <circle class="engine-combustion" cx="50" cy="50" r="17"/>
      <circle class="engine-flywheel-rim" cx="50" cy="50" r="16"/>
      <g class="engine-flywheel">
        <path d="M50 37v8M61.3 56.5l-6.9-4M38.7 56.5l6.9-4"/>
        <circle cx="50" cy="50" r="7"/>
        <circle class="engine-crankpin" cx="50" cy="39" r="2.5"/>
      </g>
      <circle class="engine-ignition" cx="50" cy="50" r="4"/>
      ${[[20, 20], [80, 20], [80, 80], [20, 80]].map(([x, y]) => `<g class="engine-bolt">
        <circle cx="${x}" cy="${y}" r="3.2"/>
        <path d="M${x - 1.4} ${y + 1.4}l2.8-2.8"/>
      </g>`).join("")}
    </svg>`;

    const embers = document.createElement("div");
    embers.className = "engine-embers";
    embers.setAttribute("aria-hidden", "true");
    const waves = Array.from({ length: 2 }, () => {
      const wave = document.createElement("i");
      wave.className = "engine-ripple";
      wave.innerHTML = `<svg class="engine-flame-licks" viewBox="-100 -100 200 200" focusable="false">
        <path class="engine-flame-tongue" fill-rule="evenodd" d="${flameOutline}"/>
      </svg>`;
      embers.appendChild(wave);
      return wave;
    });
    // Reuse particles instead of creating nodes and animationend listeners each pulse.
    const sparks = Array.from({ length: 20 }, (_, i) => {
      const spark = document.createElement("i");
      spark.className = i % 4 === 0 ? "engine-ember is-spark" : "engine-ember";
      spark.style.setProperty("--s", String(0.65 + (i * 7 % 11) / 14));
      embers.appendChild(spark);
      return spark;
    });
    el.append(embers, machine);
    return {
      machine, embers, waves, sparks,
      pistons: [...machine.querySelectorAll(".engine-piston")],
      flywheel: machine.querySelector(".engine-flywheel"),
      lastTime: null,
      phase: (Number(id) || 0) * 0.17 % 1,
      elapsed: 0,
    };
  }

  window.lookFX.engine = {
    unmount(el) {
      if (!el) return;
      el.querySelector(":scope > .engine-machine")?.remove();
      el.querySelector(":scope > .engine-embers")?.remove();
      for (const key of ["core", "weak", "frail", "heat", "pulse", "flare", "pixel"]) {
        el.style.removeProperty(`--engine-${key}`);
      }
      el.removeAttribute("data-engine-frail");
      el.removeAttribute("data-engine-gear");
      delete el._engineVisual;
    },
    tick(el, u, ctx) {
      if (!el || !el.classList.contains("look-engine")) return;
      const st = window.engineHeat[u.id] || { gear: 1, weak: 0, frail: false };
      const gear = Math.max(1, Math.min(6, st.gear || 1));
      const heat = (gear - 1) / 5;
      const visual = el._engineVisual || (el._engineVisual = mount(el, u.id));
      const scale = ctx?.scale || 1;
      const now = Number.isFinite(ctx?.time) ? ctx.time : (ctx?.now ?? performance.now()) / 1000;
      const dt = visual.lastTime == null ? 0 : Math.max(0, now - visual.lastTime);
      visual.lastTime = now;
      // Match time keeps the mechanism and fire in sync with pause and video export.
      if (!st.frail) {
        visual.phase = (visual.phase + dt * (1.1 + gear * 0.38)) % 1;
        visual.elapsed += dt;
      }
      const angle = visual.phase * Math.PI * 2;
      const pulse = (1 + Math.cos(angle * 2)) / 2;
      el.style.setProperty("--engine-core", String(gear / 6));
      el.style.setProperty("--engine-weak", String(st.frail ? 0 : Math.max(0, Math.min(1, st.weak || 0))));
      el.style.setProperty("--engine-frail", st.frail ? "1" : "0");
      el.style.setProperty("--engine-heat", `${heat * 100}%`);
      el.style.setProperty("--engine-pulse", String(st.frail ? 0 : pulse));
      el.style.setProperty("--engine-flare", String(st.frail ? 0.12 : 0.25 + heat * 0.6 + pulse * 0.15));
      el.style.setProperty("--engine-pixel", `${scale}px`);
      el.dataset.engineFrail = st.frail ? "1" : "0";
      el.dataset.engineGear = String(gear);

      visual.pistons.forEach((piston, i) => {
        const stroke = (1 + Math.cos(angle + i * Math.PI)) * 2.8;
        piston.setAttribute("transform", `translate(0 ${stroke.toFixed(3)})`);
      });
      visual.flywheel.setAttribute("transform", `rotate(${(visual.phase * 360).toFixed(3)} 50 50)`);

      const vis = Math.max(0, u.vision || 0);
      const from = Math.max(8, (u.radius || 18) * scale);
      const reach = Math.max(from + 8, vis * scale);
      visual.embers.style.opacity = st.frail || vis < 1 ? "0" : "1";
      visual.embers.style.setProperty("--r", `${reach * 2}px`);
      if (st.frail || vis < 1) return;

      visual.waves.forEach((wave, i) => {
        const t = (visual.elapsed / 1.1 + i * 0.5) % 1;
        const radius = from + (reach - from) * (1 - Math.pow(1 - t, 1.7));
        wave.style.transform = `translate(-50%, -50%) scale(${radius / reach}) rotate(${i * 83 + t * 18}deg)`;
        wave.style.opacity = String(Math.sin(t * Math.PI) * (0.3 + heat * 0.3));
      });
      const count = 8 + gear * 2;
      const life = 1.05 - heat * 0.2;
      visual.sparks.forEach((spark, i) => {
        if (i >= count) {
          spark.style.opacity = "0";
          return;
        }
        const age = visual.elapsed / life + i * 0.618;
        const t = age % 1;
        const distance = from + (reach - from) * (1 - Math.pow(1 - t, 1.3));
        const rotation = i * 137.508 + Math.floor(age) * 23;
        spark.style.transform = `rotate(${rotation}deg) translateY(${-distance}px) scale(${0.45 + Math.sin(t * Math.PI) * 0.65})`;
        spark.style.opacity = String(Math.sin(t * Math.PI) * (0.45 + heat * 0.45));
      });
    },
  };
})();
