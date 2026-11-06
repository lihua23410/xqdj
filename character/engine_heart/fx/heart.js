(function () {
  const hooks = window.lookFX = window.lookFX || {};
  const base = document.currentScript?.dataset.pack || "/ball/内燃机(泵之心)";
  const voices = new Map();
  for (const [name,file,volume] of [["heart-beat","beat.wav",.5],["heart-equip","equip.wav",.35],["heart-block","equip.wav",.25]]) {
    arena.registerShot(name,(fx,ctx={}) => {
      if (ctx.visualOnly) return;
      let pool = voices.get(name);
      if (!pool) { pool=Array.from({length:3},()=>arena.voice(new Audio(`${base}/fx/${file}`),{mode:arena.SOUND_MODE.NORMAL,priority:205,volume}));pool.next=0;voices.set(name,pool); }
      const v=pool[pool.next++%pool.length];v.stop();v.reset();v.play();
    });
  }
  arena.registerShot("heart-state",()=>{});
  const body=`<svg viewBox="-52 -54 104 108" aria-label="黑铁黄铜蒸汽泵心" role="img">
  <defs>
    

    <linearGradient id="ph-metal" x1="0" y1="0" x2="1" y2=".8">
      <stop stop-color="#536064"/>
      <stop offset=".16" stop-color="#303b40"/>
      <stop offset=".55" stop-color="#192327"/>
      <stop offset="1" stop-color="#080f13"/>
    </linearGradient>
    

    <radialGradient id="ph-body" cx=".32" cy=".22" r=".88">
      <stop stop-color="#354247"/>
      <stop offset=".4" stop-color="#162125"/>
      <stop offset="1" stop-color="#050b0f"/>
    </radialGradient>
    

    <linearGradient id="ph-brass" x1=".1" y1="0" x2=".85" y2="1">
      <stop stop-color="#785622"/>
      <stop offset=".24" stop-color="#d7ad60"/>
      <stop offset=".42" stop-color="#f9e0a1"/>
      <stop offset=".6" stop-color="#c3984d"/>
      <stop offset="1" stop-color="#73501e"/>
    </linearGradient>
    

    <linearGradient id="ph-pipeV" x1="0" y1="0" x2="1" y2="0">
      <stop stop-color="#76511e"/>
      <stop offset=".18" stop-color="#cda056"/>
      <stop offset=".36" stop-color="#f7d68d"/>
      <stop offset=".46" stop-color="#fff0bd"/>
      <stop offset=".65" stop-color="#dbb46b"/>
      <stop offset="1" stop-color="#8b622c"/>
    </linearGradient>
    

    <linearGradient id="ph-pipeH" x1="0" y1="0" x2="0" y2="1">
      <stop stop-color="#76511e"/>
      <stop offset=".18" stop-color="#cda056"/>
      <stop offset=".36" stop-color="#f7d68d"/>
      <stop offset=".46" stop-color="#fff0bd"/>
      <stop offset=".65" stop-color="#dbb46b"/>
      <stop offset="1" stop-color="#8b622c"/>
    </linearGradient>
    

    <linearGradient id="ph-elbow" x1="0" y1="0" x2=".8" y2="1">
      <stop stop-color="#f3d99d"/>
      <stop offset=".35" stop-color="#d6b06d"/>
      <stop offset=".7" stop-color="#b38744"/>
      <stop offset="1" stop-color="#785222"/>
    </linearGradient>
    

    <radialGradient id="ph-fireglass" cx=".5" cy=".95" r=".96">
      <stop stop-color="#ffce7f"/>
      <stop offset=".24" stop-color="#d78939"/>
      <stop offset=".58" stop-color="#7a3e17"/>
      <stop offset="1" stop-color="#170f0a"/>
    </radialGradient>
    

    <radialGradient id="ph-heat">
      <stop stop-color="#f5aa4f" stop-opacity=".45"/>
      <stop offset="1" stop-color="#f5aa4f" stop-opacity="0"/>
    </radialGradient>
    

    <clipPath id="ph-eye">
      <circle r="6.3"/>
    </clipPath>
    

    <clipPath id="ph-heart">
      <path d="M-23-14Q-14-21-3-18Q10-23 20-13Q28-4 23 10Q17 26 3 34Q-9 29-17 19Q-24 8-23-14Z"/>
    </clipPath>
    

    <clipPath id="ph-shell-crown">
      <rect x="-36" y="-24" width="72" height="18"/>
    </clipPath>
    

    <clipPath id="ph-shell-chest">
      <rect x="-36" y="-5" width="72" height="19"/>
    </clipPath>
    

    <clipPath id="ph-shell-drive">
      <rect x="-36" y="15.5" width="72" height="22"/>
    </clipPath>
    

  </defs>
  <g class="ph-machine" transform="rotate(-7)">
    <g class="ph-modules-rear"/>
    <g class="ph-core">
      <path d="M-21-13Q-10-19-3-16Q10-20 19-12Q25-3 20 10Q14 24 3 31Q-8 26-16 17Q-23 6-21-13Z" fill="#050c10" stroke="#02070a" stroke-width="0.6"/>
      <path d="M-9-16V21M13-16V24" fill="none" stroke="#69756b" stroke-width="1.1"/>
      <rect x="-11" y="-8" width="4" height="5" rx="0.7" fill="#111b1e" stroke="#080e11" stroke-width="0.7"/>
      <rect x="11" y="12" width="4" height="6" rx="0.7" fill="#111b1e" stroke="#080e11" stroke-width="0.7"/>
      <g class="ph-motion-crown">
        <g clip-path="url(#ph-shell-crown)">
          <path d="M-23-14Q-14-21-3-18Q10-23 20-13Q28-4 23 10Q17 26 3 34Q-9 29-17 19Q-24 8-23-14Z" fill="url(#ph-body)" stroke="#02070b" stroke-width="1.8"/>
          <g clip-path="url(#ph-heart)">
            <path d="M-10-23-7 13 0 34M13-20 16 17 9 33" fill="none" stroke="#090f13" stroke-width="2.4"/>
            <path d="M-11-23-8 13-1 34M12-20 15 17 8 33" fill="none" stroke="#66736f" stroke-width="0.55"/>
            <path d="M-25-7H-12L-7 18-14 24-25 11Z" fill="#3c494c" stroke="none" stroke-width="0" opacity=".16"/>
            <path d="M8-20H26V14L13 29Z" fill="#03080c" stroke="none" stroke-width="0" opacity=".3"/>
            <g fill="#bec4b6" stroke="#11191c" stroke-width=".3">
              <circle cx="-11" cy="-13" r="0.55"/>
              <path d="M-11.275 -13h0.55" fill="none" stroke="#263332" stroke-width="0.25"/>
              <circle cx="-10" cy="-1" r="0.55"/>
              <path d="M-10.275 -1h0.55" fill="none" stroke="#263332" stroke-width="0.25"/>
              <circle cx="-9" cy="12" r="0.55"/>
              <path d="M-9.275 12h0.55" fill="none" stroke="#263332" stroke-width="0.25"/>
              <circle cx="-4" cy="25" r="0.55"/>
              <path d="M-4.275 25h0.55" fill="none" stroke="#263332" stroke-width="0.25"/>
              <circle cx="13" cy="-12" r="0.55"/>
              <path d="M12.725 -12h0.55" fill="none" stroke="#263332" stroke-width="0.25"/>
              <circle cx="15" cy="1" r="0.55"/>
              <path d="M14.725 1h0.55" fill="none" stroke="#263332" stroke-width="0.25"/>
              <circle cx="14" cy="14" r="0.55"/>
              <path d="M13.725 14h0.55" fill="none" stroke="#263332" stroke-width="0.25"/>
              <circle cx="9" cy="26" r="0.55"/>
              <path d="M8.725 26h0.55" fill="none" stroke="#263332" stroke-width="0.25"/>
            </g>
          </g>
          <path d="M-22-14Q-14-20-3-18M2 33Q-7 29-13 23" fill="none" stroke="#81908a" stroke-width="0.6"/>
          <path d="M-23-10V7" fill="none" stroke="#566660" stroke-width="0.8"/>
        </g>
      </g>
      <g class="ph-motion-chest">
        <g clip-path="url(#ph-shell-chest)">
          <path d="M-23-14Q-14-21-3-18Q10-23 20-13Q28-4 23 10Q17 26 3 34Q-9 29-17 19Q-24 8-23-14Z" fill="url(#ph-body)" stroke="#02070b" stroke-width="1.8"/>
          <g clip-path="url(#ph-heart)">
            <path d="M-10-23-7 13 0 34M13-20 16 17 9 33" fill="none" stroke="#090f13" stroke-width="2.4"/>
            <path d="M-11-23-8 13-1 34M12-20 15 17 8 33" fill="none" stroke="#66736f" stroke-width="0.55"/>
            <path d="M-25-7H-12L-7 18-14 24-25 11Z" fill="#3c494c" stroke="none" stroke-width="0" opacity=".16"/>
            <path d="M8-20H26V14L13 29Z" fill="#03080c" stroke="none" stroke-width="0" opacity=".3"/>
            <g fill="#bec4b6" stroke="#11191c" stroke-width=".3">
              <circle cx="-11" cy="-13" r="0.55"/>
              <path d="M-11.275 -13h0.55" fill="none" stroke="#263332" stroke-width="0.25"/>
              <circle cx="-10" cy="-1" r="0.55"/>
              <path d="M-10.275 -1h0.55" fill="none" stroke="#263332" stroke-width="0.25"/>
              <circle cx="-9" cy="12" r="0.55"/>
              <path d="M-9.275 12h0.55" fill="none" stroke="#263332" stroke-width="0.25"/>
              <circle cx="-4" cy="25" r="0.55"/>
              <path d="M-4.275 25h0.55" fill="none" stroke="#263332" stroke-width="0.25"/>
              <circle cx="13" cy="-12" r="0.55"/>
              <path d="M12.725 -12h0.55" fill="none" stroke="#263332" stroke-width="0.25"/>
              <circle cx="15" cy="1" r="0.55"/>
              <path d="M14.725 1h0.55" fill="none" stroke="#263332" stroke-width="0.25"/>
              <circle cx="14" cy="14" r="0.55"/>
              <path d="M13.725 14h0.55" fill="none" stroke="#263332" stroke-width="0.25"/>
              <circle cx="9" cy="26" r="0.55"/>
              <path d="M8.725 26h0.55" fill="none" stroke="#263332" stroke-width="0.25"/>
            </g>
          </g>
          <path d="M-22-14Q-14-20-3-18M2 33Q-7 29-13 23" fill="none" stroke="#81908a" stroke-width="0.6"/>
          <path d="M-23-10V7" fill="none" stroke="#566660" stroke-width="0.8"/>
        </g>
      </g>
      <g class="ph-motion-drive">
        <g clip-path="url(#ph-shell-drive)">
          <path d="M-23-14Q-14-21-3-18Q10-23 20-13Q28-4 23 10Q17 26 3 34Q-9 29-17 19Q-24 8-23-14Z" fill="url(#ph-body)" stroke="#02070b" stroke-width="1.8"/>
          <g clip-path="url(#ph-heart)">
            <path d="M-10-23-7 13 0 34M13-20 16 17 9 33" fill="none" stroke="#090f13" stroke-width="2.4"/>
            <path d="M-11-23-8 13-1 34M12-20 15 17 8 33" fill="none" stroke="#66736f" stroke-width="0.55"/>
            <path d="M-25-7H-12L-7 18-14 24-25 11Z" fill="#3c494c" stroke="none" stroke-width="0" opacity=".16"/>
            <path d="M8-20H26V14L13 29Z" fill="#03080c" stroke="none" stroke-width="0" opacity=".3"/>
            <g fill="#bec4b6" stroke="#11191c" stroke-width=".3">
              <circle cx="-11" cy="-13" r="0.55"/>
              <path d="M-11.275 -13h0.55" fill="none" stroke="#263332" stroke-width="0.25"/>
              <circle cx="-10" cy="-1" r="0.55"/>
              <path d="M-10.275 -1h0.55" fill="none" stroke="#263332" stroke-width="0.25"/>
              <circle cx="-9" cy="12" r="0.55"/>
              <path d="M-9.275 12h0.55" fill="none" stroke="#263332" stroke-width="0.25"/>
              <circle cx="-4" cy="25" r="0.55"/>
              <path d="M-4.275 25h0.55" fill="none" stroke="#263332" stroke-width="0.25"/>
              <circle cx="13" cy="-12" r="0.55"/>
              <path d="M12.725 -12h0.55" fill="none" stroke="#263332" stroke-width="0.25"/>
              <circle cx="15" cy="1" r="0.55"/>
              <path d="M14.725 1h0.55" fill="none" stroke="#263332" stroke-width="0.25"/>
              <circle cx="14" cy="14" r="0.55"/>
              <path d="M13.725 14h0.55" fill="none" stroke="#263332" stroke-width="0.25"/>
              <circle cx="9" cy="26" r="0.55"/>
              <path d="M8.725 26h0.55" fill="none" stroke="#263332" stroke-width="0.25"/>
            </g>
          </g>
          <path d="M-22-14Q-14-20-3-18M2 33Q-7 29-13 23" fill="none" stroke="#81908a" stroke-width="0.6"/>
          <path d="M-23-10V7" fill="none" stroke="#566660" stroke-width="0.8"/>
        </g>
      </g>
      <g class="ph-motion-chest">
        <path d="M-21,-12 -21,7" fill="none" stroke="#120f09" stroke-width="7.0" stroke-linecap="square" stroke-linejoin="miter"/>
        <rect x="-23.7" y="-14.7" width="5.4" height="24.4" rx="0.25" fill="url(#ph-pipeV)" stroke="none" stroke-width="0"/>
        <rect x="-24.618000000000002" y="-12.3" width="7.236000000000001" height="2.6" rx="0.7" fill="url(#ph-brass)" stroke="#5c421c" stroke-width="0.5"/>
        <path d="M5,-15 5,-8" fill="none" stroke="#120f09" stroke-width="4.7" stroke-linecap="square" stroke-linejoin="miter"/>
        <rect x="3.45" y="-16.55" width="3.1" height="10.1" rx="0.25" fill="url(#ph-pipeV)" stroke="none" stroke-width="0"/>
      </g>
      <g class="ph-motion-drive">
        <path d="M-21,4 -21,15 12,15 12,23" fill="none" stroke="#120f09" stroke-width="7.0" stroke-linecap="square" stroke-linejoin="miter"/>
        <rect x="-23.7" y="1.2999999999999998" width="5.4" height="16.4" rx="0.25" fill="url(#ph-pipeV)" stroke="none" stroke-width="0"/>
        <rect x="-23.7" y="12.3" width="38.4" height="5.4" rx="0.25" fill="url(#ph-pipeH)" stroke="none" stroke-width="0"/>
        <rect x="9.3" y="12.3" width="5.4" height="13.4" rx="0.25" fill="url(#ph-pipeV)" stroke="none" stroke-width="0"/>
        <rect x="-24.402" y="11.597999999999999" width="6.804" height="6.804" rx="0.65" fill="url(#ph-elbow)" stroke="#63481f" stroke-width="0.55"/>
        <path d="M-23.43 13.272h4.158" fill="none" stroke="#ffdf9b" stroke-width="0.45"/>
        <rect x="8.597999999999999" y="11.597999999999999" width="6.804" height="6.804" rx="0.65" fill="url(#ph-elbow)" stroke="#63481f" stroke-width="0.55"/>
        <path d="M9.57 13.272h4.158" fill="none" stroke="#ffdf9b" stroke-width="0.45"/>
        <rect x="-24.618000000000002" y="3.7" width="7.236000000000001" height="2.6" rx="0.7" fill="url(#ph-brass)" stroke="#5c421c" stroke-width="0.5"/>
        <rect x="-24.618000000000002" y="12.7" width="7.236000000000001" height="2.6" rx="0.7" fill="url(#ph-brass)" stroke="#5c421c" stroke-width="0.5"/>
        <rect x="-15.3" y="11.382" width="2.6" height="7.236000000000001" rx="0.7" fill="url(#ph-brass)" stroke="#5c421c" stroke-width="0.5"/>
        <rect x="7.7" y="11.382" width="2.6" height="7.236000000000001" rx="0.7" fill="url(#ph-brass)" stroke="#5c421c" stroke-width="0.5"/>
        <rect x="8.382" y="20.7" width="7.236000000000001" height="2.6" rx="0.7" fill="url(#ph-brass)" stroke="#5c421c" stroke-width="0.5"/>
      </g>
      <g class="ph-motion-crown">
        <path d="M-8,-32 -8,-24 5,-24 5,-12" fill="none" stroke="#120f09" stroke-width="7.0" stroke-linecap="square" stroke-linejoin="miter"/>
        <rect x="-10.7" y="-34.7" width="5.4" height="13.4" rx="0.25" fill="url(#ph-pipeV)" stroke="none" stroke-width="0"/>
        <rect x="-10.7" y="-26.7" width="18.4" height="5.4" rx="0.25" fill="url(#ph-pipeH)" stroke="none" stroke-width="0"/>
        <rect x="2.3" y="-26.7" width="5.4" height="17.4" rx="0.25" fill="url(#ph-pipeV)" stroke="none" stroke-width="0"/>
        <rect x="-11.402000000000001" y="-27.402" width="6.804" height="6.804" rx="0.65" fill="url(#ph-elbow)" stroke="#63481f" stroke-width="0.55"/>
        <path d="M-10.43 -25.728h4.158" fill="none" stroke="#ffdf9b" stroke-width="0.45"/>
        <rect x="1.5979999999999999" y="-27.402" width="6.804" height="6.804" rx="0.65" fill="url(#ph-elbow)" stroke="#63481f" stroke-width="0.55"/>
        <path d="M2.57 -25.728h4.158" fill="none" stroke="#ffdf9b" stroke-width="0.45"/>
        <rect x="-11.618" y="-27.3" width="7.236000000000001" height="2.6" rx="0.7" fill="url(#ph-brass)" stroke="#5c421c" stroke-width="0.5"/>
        <rect x="1.3819999999999997" y="-16.3" width="7.236000000000001" height="2.6" rx="0.7" fill="url(#ph-brass)" stroke="#5c421c" stroke-width="0.5"/>
        <rect x="-3.3" y="-27.618000000000002" width="2.6" height="7.236000000000001" rx="0.7" fill="url(#ph-brass)" stroke="#5c421c" stroke-width="0.5"/>
        <g data-socket="chimney-cap"><path d="M-12-32H-4L-1-38H-15Z" fill="url(#ph-elbow)" stroke="#3a2e18" stroke-width="0.8"/>
        <rect x="-16" y="-39.5" width="16" height="2.5" rx="1" fill="url(#ph-brass)" stroke="#1d1a11" stroke-width="0.7"/></g>
      </g>
      <g class="ph-motion-crown">
        <g data-socket="crown-bypass">
          <path d="M12,-12 12,-20 18,-20 18,-13" fill="none" stroke="#120f09" stroke-width="5.300000000000001" stroke-linecap="square" stroke-linejoin="miter"/>
          <rect x="10.15" y="-21.85" width="3.7" height="11.7" rx="0.25" fill="url(#ph-pipeV)" stroke="none" stroke-width="0"/>
          <rect x="10.15" y="-21.85" width="9.7" height="3.7" rx="0.25" fill="url(#ph-pipeH)" stroke="none" stroke-width="0"/>
          <rect x="16.15" y="-21.85" width="3.7" height="10.7" rx="0.25" fill="url(#ph-pipeV)" stroke="none" stroke-width="0"/>
          <rect x="9.669" y="-22.331" width="4.662" height="4.662" rx="0.65" fill="url(#ph-elbow)" stroke="#63481f" stroke-width="0.55"/>
          <path d="M10.335 -21.184h2.849" fill="none" stroke="#ffdf9b" stroke-width="0.45"/>
          <rect x="15.669" y="-22.331" width="4.662" height="4.662" rx="0.65" fill="url(#ph-elbow)" stroke="#63481f" stroke-width="0.55"/>
          <path d="M16.335 -21.184h2.849" fill="none" stroke="#ffdf9b" stroke-width="0.45"/>
          <rect x="9.521" y="-19.3" width="4.958" height="2.6" rx="0.7" fill="url(#ph-brass)" stroke="#5c421c" stroke-width="0.5"/>
          <rect x="15.521" y="-16.3" width="4.958" height="2.6" rx="0.7" fill="url(#ph-brass)" stroke="#5c421c" stroke-width="0.5"/>
        </g>
      </g>
      <g class="ph-motion-chest">
        <g data-socket="service-bypass">
          <path d="M16,-1 22,-1 22,10 16,10" fill="none" stroke="#120f09" stroke-width="5.2" stroke-linecap="square" stroke-linejoin="miter"/>
          <rect x="14.2" y="-2.8" width="9.6" height="3.6" rx="0.25" fill="url(#ph-pipeH)" stroke="none" stroke-width="0"/>
          <rect x="20.2" y="-2.8" width="3.6" height="14.6" rx="0.25" fill="url(#ph-pipeV)" stroke="none" stroke-width="0"/>
          <rect x="14.2" y="8.2" width="9.6" height="3.6" rx="0.25" fill="url(#ph-pipeH)" stroke="none" stroke-width="0"/>
          <rect x="19.732" y="-3.2680000000000002" width="4.5360000000000005" height="4.5360000000000005" rx="0.65" fill="url(#ph-elbow)" stroke="#63481f" stroke-width="0.55"/>
          <path d="M20.38 -2.152h2.772" fill="none" stroke="#ffdf9b" stroke-width="0.45"/>
          <rect x="19.732" y="7.731999999999999" width="4.5360000000000005" height="4.5360000000000005" rx="0.65" fill="url(#ph-elbow)" stroke="#63481f" stroke-width="0.55"/>
          <path d="M20.38 8.848h2.772" fill="none" stroke="#ffdf9b" stroke-width="0.45"/>
          <rect x="19.588" y="-1.3" width="4.824000000000001" height="2.6" rx="0.7" fill="url(#ph-brass)" stroke="#5c421c" stroke-width="0.5"/>
          <rect x="19.588" y="7.7" width="4.824000000000001" height="2.6" rx="0.7" fill="url(#ph-brass)" stroke="#5c421c" stroke-width="0.5"/>
        </g>
      </g>
      <g class="ph-motion-drive">
        <g data-socket="lower-blank">
          <path d="M-14,15 -14,23 -5,23" fill="none" stroke="#120f09" stroke-width="5.300000000000001" stroke-linecap="square" stroke-linejoin="miter"/>
          <rect x="-15.85" y="13.15" width="3.7" height="11.7" rx="0.25" fill="url(#ph-pipeV)" stroke="none" stroke-width="0"/>
          <rect x="-15.85" y="21.15" width="12.7" height="3.7" rx="0.25" fill="url(#ph-pipeH)" stroke="none" stroke-width="0"/>
          <rect x="-16.331" y="20.669" width="4.662" height="4.662" rx="0.65" fill="url(#ph-elbow)" stroke="#63481f" stroke-width="0.55"/>
          <path d="M-15.665 21.816h2.849" fill="none" stroke="#ffdf9b" stroke-width="0.45"/>
          <rect x="-16.479" y="19.7" width="4.958" height="2.6" rx="0.7" fill="url(#ph-brass)" stroke="#5c421c" stroke-width="0.5"/>
          <rect x="-6" y="21" width="3" height="4" rx="0.5" fill="url(#ph-elbow)" stroke="#080e11" stroke-width="0.7"/>
        </g>
      </g>
      <g class="ph-motion-drive">
        <g data-socket="drive-cover">
          <circle cx="12" cy="23" r="5" fill="url(#ph-brass)" stroke="#0b1216" stroke-width="0.7"/>
          <circle cx="12" cy="23" r="3.8" fill="#142125" stroke="#0b1216" stroke-width="0.7"/>
          <g transform="translate(12 23)">
            <g class="ph-cog">
              <path d="M-3 0H3M0-3V3" fill="none" stroke="#c49d58" stroke-width="0.9"/>
              <circle cx="0" cy="0" r="1" fill="#c6ad77" stroke="#0b1216" stroke-width="0.7"/>
            </g>
          </g>
          <circle cx="5" cy="29" r="3.1" fill="#1d292e" stroke="url(#ph-brass)" stroke-width="1.2"/>
        </g>
      </g>
      <g class="ph-motion-drive">
        <g data-socket="fuel-valve">
          <g transform="translate(-5 15)">
            <g class="ph-steam-valve">
              <circle cx="0" cy="0" r="2.8" fill="#101a1d" stroke="url(#ph-brass)" stroke-width="1.1"/>
              <path d="M-2.8 0H2.8M0 -2.8V2.8" fill="none" stroke="#e8bd6e" stroke-width="0.65"/>
              <circle cx="0" cy="0" r="0.8" fill="#c49a50" stroke="#0b1216" stroke-width="0.7"/>
            </g>
          </g>
        </g>
      </g>
      <g class="ph-motion-keel">
        <g data-socket="keel-cap">
          <path d="M-1 32H6L4 37Q3 39 1 37Z" fill="url(#ph-brass)" stroke="#1a170f" stroke-width="0.7"/>
        </g>
      </g>
      <g class="ph-motion-chest">
        <circle cx="2" cy="2" r="14" fill="url(#ph-heat)" stroke="none" stroke-width="0" class="ph-glow"/>
        <g transform="translate(2 2)" class="ph-furnace">
          <circle cx="0" cy="0" r="9" fill="#070c10" stroke="#020508" stroke-width="1"/>
          <circle cx="0" cy="0" r="8.2" fill="url(#ph-brass)" stroke="#0b1216" stroke-width="0.7"/>
          <circle cx="0" cy="0" r="6.7" fill="#1a150d" stroke="#49351b" stroke-width="0.6"/>
          <g clip-path="url(#ph-eye)">
            <circle cx="0" cy="0" r="6.3" fill="url(#ph-fireglass)" stroke="none" stroke-width="0"/>
            <g class="ph-furnace-fire">
              <ellipse cy="4" rx="5.6" ry="2.7" fill="#eba14b"/>
              <ellipse cy="4.5" rx="2.9" ry="1.4" fill="#ffe0a3"/>
            </g>
            <path d="M-5-2Q-4-5 1-5L-5 3Z" fill="#fff4cf" stroke="none" stroke-width="0" opacity=".13"/>
          </g>
          <g fill="#bec4b6" stroke="#11191c" stroke-width=".3">
            <circle cx="0.0" cy="7.7" r="0.5"/>
            <path d="M-0.25 7.7h0.5" fill="none" stroke="#263332" stroke-width="0.25"/>
            <circle cx="5.44" cy="5.44" r="0.5"/>
            <path d="M5.19 5.44h0.5" fill="none" stroke="#263332" stroke-width="0.25"/>
            <circle cx="7.7" cy="0.0" r="0.5"/>
            <path d="M7.45 0.0h0.5" fill="none" stroke="#263332" stroke-width="0.25"/>
            <circle cx="5.44" cy="-5.44" r="0.5"/>
            <path d="M5.19 -5.44h0.5" fill="none" stroke="#263332" stroke-width="0.25"/>
            <circle cx="0.0" cy="-7.7" r="0.5"/>
            <path d="M-0.25 -7.7h0.5" fill="none" stroke="#263332" stroke-width="0.25"/>
            <circle cx="-5.44" cy="-5.44" r="0.5"/>
            <path d="M-5.69 -5.44h0.5" fill="none" stroke="#263332" stroke-width="0.25"/>
            <circle cx="-7.7" cy="-0.0" r="0.5"/>
            <path d="M-7.95 -0.0h0.5" fill="none" stroke="#263332" stroke-width="0.25"/>
            <circle cx="-5.44" cy="5.44" r="0.5"/>
            <path d="M-5.69 5.44h0.5" fill="none" stroke="#263332" stroke-width="0.25"/>
          </g>
        </g>
        <g transform="translate(-13 -4)">
          <circle cx="0" cy="0" r="4.7" fill="url(#ph-brass)" stroke="#14170f" stroke-width="0.8"/>
          <circle cx="0" cy="0" r="3.6" fill="#dacb9b" stroke="#665432" stroke-width="0.45"/>
          <path d="M-2.7 0H-2M0-2.7V-2M2.7 0H2M-2-2-1.5-1.5M2-2 1.5-1.5" fill="none" stroke="#6a6147" stroke-width="0.4"/>
          <g class="ph-pressure-needle">
            <path d="M0 1V-2.6" fill="none" stroke="#743d20" stroke-width="0.6"/>
          </g>
          <circle cx="0" cy="0" r="0.55" fill="#51442e" stroke="#0b1216" stroke-width="0.7"/>
        </g>
      </g>
      <g class="ph-motion-crown">
        <g class="ph-steam" fill="#ccd5ca">
          <circle cx="-8" cy="-42" r="1.4" opacity=".5"/>
          <circle cx="-10" cy="-45" r="2" opacity=".3"/>
          <circle cx="-6" cy="-47" r="1.5" opacity=".17"/>
        </g>
      </g>
    </g>
    <g class="ph-modules-front"/>
  </g>
</svg>`;
  function get(id) { return document.getElementById(`pump-floor-${id}`)?._pump?.data; }
  const partDefs=body.match(/<defs>[\s\S]*?<\/defs>/)[0];
  function renderPart(el,type,id) {
    const def=window.pumpParts?.[type];if(!def)return;
    if(el.dataset.partType===String(type)&&el.querySelector("svg"))return;
    el.dataset.partType=String(type);el.title=def.name;
    el.innerHTML=scopeSVG(`<svg xmlns="http://www.w3.org/2000/svg" role="img" aria-label="${def.name}完整零件">${partDefs}${def.svg}</svg>`,id);
    const svg=el.querySelector("svg"),box=svg.querySelector("[data-part]").getBBox();
    const side=Math.max(box.width,box.height)+6;
    svg.setAttribute("viewBox",`${box.x+box.width/2-side/2} ${box.y+box.height/2-side/2} ${side} ${side}`);
  }
  function groundParts(st,el,d,ctx,minX,maxY,w,h) {
    if(!st.partsLayer&&!d.parts.some(p=>!p.equipped))return;
    if(!st.partsLayer){st.partsLayer=document.createElement("div");st.partsLayer.className="pump-ground-parts";el.appendChild(st.partsLayer);st.partNodes=new Map();}
    d.parts.forEach((p,i)=>{
      let node=st.partNodes.get(i);
      if(!node){node=document.createElement("span");node.className="pump-part";st.partsLayer.appendChild(node);st.partNodes.set(i,node);}
      node.style.display=p.equipped?"none":"block";if(p.equipped)return;
      renderPart(node,p.type,`ground-${el.id}-${i}`);
      Object.assign(node.style,{left:`${(p.x-minX)/w*100}%`,top:`${(maxY-p.y)/h*100}%`,width:`${36*ctx.scale}px`,height:`${36*ctx.scale}px`,transform:`translate(-50%,-50%) rotate(${p.flying?ctx.time*12+p.type:0}rad)`});
    });
  }
  // Pickup order controls shedding only. Each part owns a permanent machine-space mount.
  function scopeSVG(markup,id) {
    return markup.replace(/id="ph-/g,`id="ph-${id}-`).replace(/url\(#ph-/g,`url(#ph-${id}-`);
  }
  // Rigid carriages close in sequence, hold pressure briefly, then slide back out.
  // This is driven by simulation phase, so pause and faster cams keep the same pose.
  function mechanicalPose(d={}) {
    function pressure(age) {
      if(age<0||age>=.24)return 0;
      const smooth=t=>t*t*(3-2*t);
      if(age<.045)return smooth(age/.045);
      if(age<.095)return 1;
      return 1-smooth((age-.095)/.145);
    }
    const pose={};
    for(const [section,lag,travel] of [["crown",0,3.2],["chest",.04,.85],["drive",.08,-2.7],["keel",.12,-4.6]]){
      let stroke=pressure((d.phase||0)-.36-lag);
      if(d.echo)stroke+=.4*pressure((d.phase||0)-.56-lag);
      pose[section]=d.suspended?0:Math.min(1.15,stroke)*travel;
    }
    return pose;
  }
  function render(el,d={},ctx={}) {
    let art=el.querySelector(":scope > .pump-body");
    const id=el.dataset.pumpID || (el.dataset.pumpID=String(ctx.id||"preview"));
    if(!art){art=document.createElement("span");art.className="pump-body";art.innerHTML=scopeSVG(body,id);el.appendChild(art);}
    const phase=d.phase||0,stack=d.stack||[],parts=new Set(stack);
    let kick=d.suspended?0:Math.exp(-Math.pow((phase-.4)/.055,2));
    if(d.echo&&!d.suspended)kick+=.4*Math.exp(-Math.pow((phase-.6)/.045,2));
    const lastContraction=d.echo&&phase>=.6?.6:.4;
    const steamAge=phase>=.4?phase-lastContraction:-1;
    const steam=d.suspended||steamAge<0?0:Math.exp(-steamAge*10);
    const time=ctx.time||0;
    const pose=mechanicalPose(d);
    for(const [section,y] of Object.entries(pose))art.style.setProperty(`--ph-${section}-y`,`${y.toFixed(4)}px`);
    art.style.setProperty("--ph-fire",String(.16+kick*.78));
    art.style.setProperty("--ph-stroke",String(kick*5));
    art.style.setProperty("--ph-wheel",`${time*8}rad`);
    art.style.setProperty("--ph-turbine",`${time*12}rad`);
    art.style.setProperty("--ph-crank",`${phase*Math.PI*2}rad`);
    art.style.setProperty("--ph-link",`${kick*7}deg`);
    art.style.setProperty("--ph-steam",String(steam*.65));
    art.style.setProperty("--ph-steam-travel",String(steamAge<0?0:Math.min(10,steamAge*26)));
    art.style.setProperty("--ph-left-jet",String((d.beats||0)%2?0:kick*.9));
    art.style.setProperty("--ph-right-jet",String((d.beats||0)%2?kick*.9:0));
    art.dataset.equipped=stack.length;
    const layoutKey=[...parts].sort((a,b)=>a-b).join(","),replaced=new Set();
    for(const t of [...parts].sort((a,b)=>a-b)){
      const def=window.pumpParts?.[t];if(!def?.mount)continue;
      const layer=art.querySelector(`.ph-modules-${def.mount.layer==="rear"?"rear":"front"}`);
      let node=art.querySelector(`[data-part="${t}"]`);
      if(!node){layer.insertAdjacentHTML("beforeend",scopeSVG(def.svg,id));node=art.querySelector(`[data-part="${t}"]`);art.dataset.layout="";}
      node.setAttribute("transform",`translate(${def.mount.x} ${def.mount.y})`);
      node.setAttribute("aria-label",`${def.name} · ${def.site}`);node.style.display="inline";
      if(art.dataset.layout!==layoutKey)layer.appendChild(node);
      for(const socket of def.replaces||[])replaced.add(socket);
    }
    for(const node of art.querySelectorAll("[data-part]"))if(!parts.has(Number(node.dataset.part)))node.style.display="none";
    for(const node of art.querySelectorAll("[data-socket]"))node.style.display=replaced.has(node.dataset.socket)?"none":"inline";
    art.dataset.layout=layoutKey;
    art.classList.toggle("has-ammo",!!d.ammo);
    return art;
  }
  function paintPulse(c,p,now) {
    const age=now-p.at,t=Math.min(1,Math.max(0,age/.42)),fade=Math.pow(1-t,1.3)*p.power;
    const radius=18+(p.radius-18)*(1-Math.pow(1-t,1.6));
    c.save();
    c.beginPath();c.arc(p.x,p.y,radius,0,Math.PI*2);c.strokeStyle=`rgba(241,179,87,${fade*.85})`;c.lineWidth=4.5*(1-t)+1;c.stroke();
    c.beginPath();c.arc(p.x,p.y,Math.max(1,radius-6),0,Math.PI*2);c.strokeStyle=`rgba(255,230,173,${fade*.65})`;c.lineWidth=1.4;c.stroke();
    for(let i=0;i<16;i++){
      const a=i*Math.PI/8+p.at*.7,inner=radius*(.55+.25*(i%3)/2),outer=inner+8+18*t;
      c.beginPath();c.moveTo(p.x+Math.cos(a)*inner,p.y+Math.sin(a)*inner);c.lineTo(p.x+Math.cos(a)*outer,p.y+Math.sin(a)*outer);
      c.strokeStyle=`rgba(230,162,70,${fade*.8})`;c.lineWidth=i%3===0?2:1;c.stroke();
    }
    if(age<.12){c.beginPath();c.arc(p.x,p.y,19+age*120,0,Math.PI*2);c.fillStyle=`rgba(255,214,145,${(1-age/.12)*.22*p.power})`;c.fill();}
    c.restore();
  }
  hooks["pump-floor"]={guide(u,ctx){
    const owner=u.ownerId;if(!owner)return;
    const id=`pump-floor-${owner}`,el=ctx.ensureGuide(id,"pump-floor");ctx.seenGuides.add(id);
    let st=el._pump;if(!st){const canvas=document.createElement("canvas");el.appendChild(canvas);st=el._pump={canvas,time:-1,pulses:[],shots:[],stamp:""};}
    if(ctx.time<st.time){st.data=null;st.pulses=[];st.shots=[];st.stamp="";}
    for(const fx of ctx.effects||[]){
      if(fx.unitId!==owner)continue;
      if(fx.name==="heart-state")st.data=fx.data;
      if(fx.name==="heart-beat"){
        const stamp=`${ctx.time}|${fx.x}|${fx.y}|${fx.amount}`;
        if(st.stamp!==stamp){st.pulses.push({at:ctx.time,x:fx.x,y:fx.y,radius:fx.vx||st.data?.radius||108,power:fx.amount||1});st.stamp=stamp;}
      }
      const visual=window.pumpAttackFX?.[fx.name];
      if(visual){
        if(visual.channel)st.shots=st.shots.filter(p=>p.visual.channel!==visual.channel);
        const key=`${ctx.time}|${fx.name}|${fx.x}|${fx.y}|${fx.vx}|${fx.vy}`;
        if(visual.duration>0&&!st.shots.some(p=>p.key===key))st.shots.push({fx,at:ctx.time,key,visual});
      }
    }
    st.pulses=st.pulses.filter(p=>ctx.time-p.at<.45);
    st.shots=st.shots.filter(p=>ctx.time-p.at<p.visual.duration);
    st.time=ctx.time;const d=st.data,heart=(ctx.units||[]).find(v=>v.id===owner);if(!d||!heart)return;
    const pts=[heart,...d.parts.filter(p=>!p.equipped),...st.pulses,...st.shots.flatMap(p=>[{x:p.fx.x,y:p.fx.y},{x:p.fx.vx,y:p.fx.vy}])],pad=Math.max(d.radius||108,42)+22;
    const minX=Math.min(...pts.map(p=>p.x))-pad,maxX=Math.max(...pts.map(p=>p.x))+pad,minY=Math.min(...pts.map(p=>p.y))-pad,maxY=Math.max(...pts.map(p=>p.y))+pad;
    const w=maxX-minX,h=maxY-minY,ratio=Math.min(2,2400/Math.max(w,h)),cw=Math.ceil(w*ratio),ch=Math.ceil(h*ratio),canvas=st.canvas;
    if(canvas.width!==cw||canvas.height!==ch){canvas.width=cw;canvas.height=ch;}
    const c=canvas.getContext("2d");c.setTransform(1,0,0,1,0,0);c.clearRect(0,0,cw,ch);c.setTransform(ratio,0,0,-ratio,-minX*ratio,maxY*ratio);
    c.save();
    if(d.outline?.length){c.beginPath();d.outline.forEach((p,i)=>{if(i)c.lineTo(p.x,p.y);else c.moveTo(p.x,p.y);});c.closePath();c.clip();}
    for(const p of st.pulses)paintPulse(c,p,ctx.time);
    for(const p of st.shots){c.save();p.visual.draw(c,p.fx,ctx.time-p.at);c.restore();}
    if(!d.suspended){
      if(d.phase<.4){for(let i=0;i<8;i++){let a=i*Math.PI/4+ctx.time*.2,r=(d.radius||108)*(1-d.phase*1.5);c.beginPath();c.arc(heart.x,heart.y,r,a,a+.12);c.strokeStyle="#cda46580";c.lineWidth=1.6;c.stroke();}}
    }
    if(d.winch>=0&&d.parts[d.winch]&&!d.parts[d.winch].equipped){const p=d.parts[d.winch];c.beginPath();c.moveTo(heart.x,heart.y);c.lineTo(p.x,p.y);c.strokeStyle="#b9ab8c";c.lineWidth=1.1;c.setLineDash([2,1]);c.stroke();c.setLineDash([]);}
    for(const p of d.parts){if(p.equipped||!p.flying)continue;c.beginPath();c.moveTo(p.x,p.y);c.lineTo(p.x-p.vx*.045,p.y-p.vy*.045);c.strokeStyle="#deaa6755";c.lineWidth=2;c.stroke();}
    for(const g of d.guards||[]){if(g.capture)continue;const r=(g.inner+g.outer)/2,a=g.angle;c.beginPath();c.arc(heart.x,heart.y,r,a-g.span/2,a+g.span/2);c.strokeStyle="#10191e";c.lineWidth=g.outer-g.inner+1;c.stroke();c.strokeStyle="#63716f";c.lineWidth=g.outer-g.inner-1;c.stroke();c.strokeStyle="#ce9562";c.lineWidth=1;c.stroke();}
    c.restore();
    Object.assign(el.style,{left:`${ctx.cx+minX*ctx.scale}px`,top:`${ctx.cy-maxY*ctx.scale}px`,width:`${w*ctx.scale}px`,height:`${h*ctx.scale}px`});
    if(d.outline?.length)el.style.clipPath=`polygon(${d.outline.map(p=>`${(p.x-minX)/w*100}% ${(maxY-p.y)/h*100}%`).join(",")})`;
    groundParts(st,el,d,ctx,minX,maxY,w,h);
  },tick(){}};
  hooks["pump-heart"]={render,renderPart,mechanicalPose,tick(el,u,ctx={}){
    render(el,get(u.id)||{},{...ctx,id:u.id});
    const hp=document.getElementById(`hp-${u.id}`);
    if(hp){hp.classList.add("pump-heart-hp");hp.style.setProperty("--ph-hp-lift",`${u.radius*(ctx.scale||1)}px`);}
  },unmount(el){
    el.querySelector(":scope > .pump-body")?.remove();
    const hp=document.getElementById(`hp-${el.dataset.pumpID}`);
    if(hp){hp.classList.remove("pump-heart-hp");hp.style.removeProperty("--ph-hp-lift");}
  }};
})();
