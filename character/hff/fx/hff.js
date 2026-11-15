window.lookFX = window.lookFX || {};

(function () {
  const arena = window.arena;
  const fishArt = "/ball/大肥鱼/fx/fish-02.png";
  let fadingHost, fishOpacity;

  function art(el) {
    let img = el.querySelector(":scope > .hff-art");
    if (!img) {
      img = document.createElement("img");
      img.className = "hff-art";
      img.alt = "";
      el.appendChild(img);
    }
    return img;
  }

  window.lookFX.hff = {
    unmount(el) {
      el?.querySelector(":scope > .hff-art")?.remove();
    },
    tick(el, u) {
      if (!el) return;
      const img = art(el);
      if (img.getAttribute("src") !== fishArt) img.src = fishArt;
      img.style.opacity = fadingHost === u?.id ? fishOpacity : "";
    },
  };

  if (!arena) return;

  const svgNS = "http://www.w3.org/2000/svg";
  const cutInImage = new Image();
  cutInImage.alt = "老实写代码，很枯燥吧";
  cutInImage.src = "/ball/大肥鱼(幻方量化)/fx/stock-cut-in.webp";
  let market, decision, settlement, parts;
  let cutIn;
  let shadow, revealLayer, revealedBall;
  let chartKey = "";
  let chartCandles = [];

  function mountMarket(hex) {
    market = document.createElement("div");
    market.className = "hff-market";
    market.setAttribute("role", "group");
    market.setAttribute("aria-label", "幻方量化行情");
    market.innerHTML = `
      <div class="hff-market-top"><span>幻方量化</span><b data-part="phase"></b></div>
      <div class="hff-market-title"><strong data-part="stock"></strong><b data-part="side"></b></div>
      <div data-part="book">
        <div class="hff-market-readout"><span data-part="metric"></span><strong data-part="net"></strong></div>
        <svg data-part="chart" viewBox="0 0 212 64" role="img" aria-label="收益 K 线"></svg>
        <div data-part="clock" class="hff-market-clock"><span data-part="time"></span></div>
      </div>`;
    parts = Object.fromEntries([...market.querySelectorAll("[data-part]")].map(el => [el.dataset.part, el]));
    decision = document.createElement("div");
    decision.className = "hff-market-decision";
    decision.setAttribute("aria-hidden", "true");
    decision.innerHTML = '<i></i><b></b>';
    hex.prepend(decision, market);
    settlement = svg("svg", { class: "fx hff-settlement-fx", "aria-hidden": "true", hidden: "" });
    cutIn = document.createElement("div");
    cutIn.className = "fx hff-stock-cutin";
    cutIn.hidden = true;
    cutIn.append(cutInImage);
    (hex.parentElement.querySelector(".over") || hex).append(settlement, cutIn);
  }

  function drawStockReveal(view, state, hex, project, ctx) {
    const opening = !!view?.preview && view.openProgress > .36;
    const liveStock = state.paper && (state.units || []).find(u => u.id === view?.hostId && u.hp > 0);
    const body = opening ? view.preview : liveStock;
    fadingHost = opening ? view.hostId : null;
    const blend = opening ? Math.min(1, (view.openProgress - .36) / .64) : 1;
    const progress = blend * blend * (3 - 2 * blend);
    fishOpacity = 1 - progress;

    if (!opening && revealedBall) {
      ctx.unmountLookFX(revealedBall);
      revealLayer.remove();
      revealedBall = revealLayer = null;
    }
    if (!body) {
      if (shadow) shadow.hidden = true;
      return;
    }
    if (!shadow) {
      shadow = document.createElement("div");
      shadow.className = "fx hff-stock-shadow";
      const fish = document.createElement("img");
      fish.src = fishArt;
      fish.alt = "";
      shadow.append(fish);
      // The projection can rise past the arena rim like the other foreground effects.
      (hex.parentElement.querySelector(".over") || hex).prepend(shadow);
    }
    const [x, y] = project(body.x, body.y);
    const diameter = body.radius * 2 * ctx.scale;
    const shadowWidth = Math.max(94 * ctx.scale, diameter * 1.72 + 32 * ctx.scale);
    const lift = shadowWidth / 2 + diameter * .17;
    shadow.hidden = false;
    shadow.style.left = `${x}px`;
    shadow.style.top = `${y - lift * progress}px`;
    shadow.style.width = `${shadowWidth}px`;
    shadow.style.opacity = .62 * progress;

    if (!opening) return;
    if (!revealedBall) {
      revealLayer = document.createElement("div");
      revealLayer.className = "hff-stock-reveal";
      revealLayer.setAttribute("aria-hidden", "true");
      revealedBall = document.createElement("div");
      revealedBall.className = "ball fighter";
      revealLayer.append(revealedBall);
      hex.append(revealLayer);
    }
    revealLayer.style.left = `${x}px`;
    revealLayer.style.top = `${y}px`;
    revealLayer.style.opacity = progress;
    revealLayer.style.setProperty("--hff-reveal-scale", .86 + .14 * progress);
    revealedBall.style.width = revealedBall.style.height = `${diameter}px`;
    revealedBall.classList.toggle("semi", Boolean(body.semi));
    if (body.semi) {
      const angle = Math.atan2(-body.faceY, body.faceX);
      revealedBall.style.setProperty("--face-ang", `${angle}rad`);
      revealedBall.style.transform = `translate(-50%, -50%) rotate(${angle}rad)`;
    } else {
      revealedBall.style.transform = "";
    }
    const look = arena.lookOf(body.kind);
    revealedBall.classList.toggle("glow", Boolean(look.glow));
    revealedBall.classList.toggle("ring", Boolean(look.ring));
    ctx.paintKind(revealedBall, body.kind);
    ctx.applyLookFX(revealedBall, body.kind);
    ctx.runLookFX(revealedBall, { ...body, id: -body.id }, {
      now: state.time * 1000, time: state.time, scale: ctx.scale, cx: ctx.cx, cy: ctx.cy,
      units: [], fxRoot: revealLayer, preview: true,
    });
  }

  function svg(tag, attrs, label) {
    const el = document.createElementNS(svgNS, tag);
    for (const [key, val] of Object.entries(attrs)) el.setAttribute(key, val);
    if (label) el.textContent = label;
    return el;
  }

  function drawCandles(candles, duration) {
    const key = JSON.stringify([candles, duration]);
    if (key === chartKey) return;
    chartKey = key;
    const peak = Math.max(1, ...candles.map(c => Math.max(c.high, -c.low)));
    const step = 10 ** Math.floor(Math.log10(peak));
    const limit = Math.ceil(peak / step) * step;
    const y = n => 32 - 23 * n / limit;
    const chart = parts.chart;
    const children = [limit, 0, -limit].flatMap(n => [
      svg("line", { x1: 0, x2: 175, y1: y(n), y2: y(n), class: n === 0 ? "zero" : "grid" }),
      svg("text", { x: 211, y: y(n) + 3, "text-anchor": "end" }, signed(n)),
    ]);
    const span = 175 / Math.max(1, Math.ceil(duration));
    chartCandles = candles.map((c, i) => ({
      x: (i + .5) * span,
      top: y(Math.max(c.open, c.close)),
      height: Math.max(1.5, Math.abs(y(c.open) - y(c.close))),
      high: y(c.high),
      low: y(c.low),
      trend: c.close > c.open ? "rise" : c.close < c.open ? "fall" : "flat",
    }));
    for (const candle of chartCandles) {
      const group = svg("g", { class: candle.trend });
      group.append(...candleGlyph(candle));
      children.push(group);
    }
    chart.replaceChildren(...children);
  }

  function candleGlyph(candle) {
    const { x, high, low, top, height } = candle;
    return [
      svg("line", { x1: x, x2: x, y1: high, y2: low }),
      svg("rect", { x: x - 4, y: top, width: 8, height, rx: 1 }),
    ];
  }

  const amount = n => n.toFixed(1).replace(/\.0$/, "");
  function signed(n) {
    const abs = amount(Math.abs(n));
    return abs === "0" ? abs : (n > 0 ? "+" : "−") + abs;
  }

  function drawSettlement(view, state, hex, project, targets) {
    const flight = view.flight || 0;
    settlement.toggleAttribute("hidden", !flight || !targets.length);
    if (!flight || !targets.length) return;

    // Convert the printed chart to arena coordinates, including the export scale.
    const frame = hex.getBoundingClientRect();
    const chart = parts.chart.getBoundingClientRect();
    const scale = frame.width / hex.clientWidth || 1;
    const chartLeft = (chart.left - frame.left) / scale;
    const chartTop = (chart.top - frame.top) / scale;
    const chartScale = chart.width / scale / parts.chart.viewBox.baseVal.width;
    const children = [];
    settlement.setAttribute("viewBox", `0 0 ${hex.clientWidth} ${hex.clientHeight}`);
    const destinations = targets.map(target => {
      const liveTarget = (state.units || []).find(u => u.id === target.id) || target;
      const point = target.impactSeq ? target : liveTarget;
      const [x, y] = project(point.x, point.y);
      return { target, liveTarget, x, y };
    });
    chartCandles.forEach((candle, i) => {
      const destination = destinations[i % destinations.length];
      if (destination.target.impactSeq) return;
      const candleY = candle.top + candle.height / 2;
      const startX = chartLeft + candle.x * chartScale;
      const startY = chartTop + candleY * chartScale;
      const dx = destination.x - startX;
      const dy = destination.y - startY;
      const distance = Math.hypot(dx, dy) || 1;
      const order = i / Math.max(1, chartCandles.length - 1);
      const delay = .32 * order;
      const travel = Math.max(0, (flight - delay) / (1 - delay));
      const t = Math.min(.985, travel * (.65 + .35 * travel));
      const spread = (.5 - order) * Math.sign(dy || 1) * 128;
      const at = p => {
        const arc = Math.sin(Math.PI * p) * spread;
        return [startX + dx * p - dy / distance * arc, startY + dy * p + dx / distance * arc];
      };
      const [tipX, tipY] = at(t);
      const [tailX, tailY] = at(Math.max(0, t - .1));
      const turn = Math.min(1, travel * 5);
      const angle = Math.atan2(tipY - tailY, tipX - tailX) * 180 / Math.PI + 90;
      const size = chartScale * (1 - .38 * turn);
      const shot = svg("g", { class: `hff-candle-shot ${candle.trend}` });
      if (travel > 0) {
        shot.append(svg("line", { x1: tailX, y1: tailY, x2: tipX, y2: tipY, class: "hff-shot-trail", opacity: turn }));
      }
      const body = svg("g", {
        class: "hff-shot-candle",
        transform: `translate(${tipX} ${tipY}) rotate(${angle * turn}) scale(${size}) translate(${-candle.x} ${-candleY})`,
      });
      body.append(...candleGlyph(candle));
      shot.append(body);
      children.push(shot);
    });
    destinations.forEach(({ target, liveTarget, x, y }) => {
      if (!target.impactSeq) return;
      const frames = state.seq - target.impactSeq;
      const age = frames / (view.impactTicks || 30);
      if (age >= 1) return;
      const group = svg("g", { class: target.self ? "hff-shot-self" : "hff-shot-enemy" });
      const ringAge = frames / 30;
      if (ringAge < 1) {
        group.append(svg("circle", {
          cx: x, cy: y, r: 12 + ringAge * 30, class: "hff-impact", opacity: 1 - ringAge,
        }));
      }
      if (typeof target.loss === "number") {
        let label = "未兑现";
        if (target.loss > 0) label = target.loss < .1 ? "兑现不足 0.1" : `兑现 −${amount(target.loss)}`;
        // Keep the label inside the arena when the target is against an edge.
        const [labelX, labelY] = project(liveTarget.x, liveTarget.y);
        const centerX = hex.clientWidth / 2;
        const centerY = hex.clientHeight / 2;
        const tx = labelX + Math.sign(centerX - labelX) * 20;
        const ty = labelY + (labelY > centerY ? -1 : 1) * (29 + age * 8);
        group.append(svg("text", { x: tx, y: ty, class: "hff-impact-text", opacity: Math.min(1, (1 - age) * 4) }, label));
      }
      children.push(group);
    });
    settlement.replaceChildren(...children);
  }

  arena.registerArenaOverlay((state, hex, project, ctx) => {
    const view = state.paperView;
    drawStockReveal(view, state, hex, project, ctx);
    if (!view) {
      if (market) {
        market.hidden = decision.hidden = true;
        cutIn.hidden = true;
        settlement.setAttribute("hidden", "");
      }
      return;
    }
    if (!market) mountMarket(hex);
    cutIn.hidden = !view.cutIn || !cutInImage.complete || !cutInImage.naturalWidth;
    settlement.setAttribute("hidden", "");
    if (!cutIn.hidden) {
      market.hidden = decision.hidden = true;
      cutIn.style.opacity = Math.min(1, view.cutIn / .1, (1 - view.cutIn) / .18);
      return;
    }
    market.hidden = false;
    const phase = view.settled ? "close" : view.openProgress ? "stare" : "paper";
    market.dataset.phase = phase;
    parts.phase.parentElement.hidden = phase !== "stare";
    parts.phase.textContent = phase === "stare" ? "盯盘 · 锁定方向" : "";
    parts.book.hidden = phase !== "paper" && phase !== "close";
    decision.hidden = phase !== "stare";
    market.style.removeProperty("--hff-opacity");
    market.style.removeProperty("--hff-ink");
    market.style.removeProperty("--hff-chart-opacity");
    market.removeAttribute("data-cue");

    parts.stock.textContent = view.stock;
    parts.side.textContent = view.long ? "▲ 看涨" : "▼ 看空";
    market.dataset.side = view.long ? "long" : "short";
    decision.dataset.side = market.dataset.side;
    if (phase === "stare") {
      const [line] = project(view.line, 0);
      const [x, y] = project(view.hostX, view.hostY);
      decision.style.setProperty("--hff-line", `${line}px`);
      decision.style.setProperty("--hff-x", `${x}px`);
      decision.style.setProperty("--hff-y", `${y}px`);
      decision.style.setProperty("--hff-scan", Math.min(1, view.openProgress * 3));
      return;
    }
    const settled = phase === "close";
    const targets = (view.targets || []).filter(Boolean);
    const awaitingResult = settled && targets.some(target => typeof target.loss !== "number");
    const hasResult = settled && targets.length > 0 && !awaitingResult;
    const net = hasResult
      ? targets.reduce((sum, target) => sum + (target.self ? -target.loss : target.loss), 0)
      : view.enemy - view.self;
    parts.metric.textContent = settled ? (awaitingResult ? "预计收益" : "本轮收益") : "当前收益";
    parts.net.textContent = signed(net);
    parts.net.dataset.result = net > 0 ? "gain" : net < 0 ? "loss" : "even";
    parts.clock.hidden = settled;
    drawCandles(view.candles || [], view.duration);
    if (settled) {
      const flight = view.flight || 0;
      if (targets.length) {
        market.dataset.cue = flight === 0 ? "freeze" : "flight";
        market.style.setProperty("--hff-chart-opacity", Math.max(0, 1 - flight * 3));
      }
      market.style.setProperty("--hff-opacity", Math.max(0, Math.min(1, (1 - (view.closeProgress || 0)) / .14)));
      drawSettlement(view, state, hex, project, targets);
    } else {
      market.style.setProperty("--hff-ink", Math.min(1, .4 + 3 * view.elapsed));
      parts.time.textContent = `${view.elapsed.toFixed(1)} / ${view.duration} 秒`;
    }
  });

  // 旧录像用逐帧的 line 特效，没有行情快照。
  arena.registerShot("line", (fx, ctx) => {
    const parent = document.getElementById("fx");
    if (!parent) return;
    let el = window.__hffLine;
    if (!el || !el.isConnected) {
      el = document.createElement("div");
      el.className = "fx hff-line";
      parent.appendChild(el);
      window.__hffLine = el;
    }
    const name = typeof fx.data === "string" ? fx.data : "";
    el.style.left = `${ctx.x}px`;
    el.style.top = "0px";
    el.style.height = `${parent.clientHeight || 0}px`;
    el.textContent = name;
    clearTimeout(window.__hffLineTimer);
    window.__hffLineTimer = setTimeout(() => {
      if (window.__hffLine) window.__hffLine.remove();
      window.__hffLine = null;
    }, 200);
  });
})();
