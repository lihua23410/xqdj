// 静电魔球的一次性特效：闪电链的一跳。
// 后端每 0.25 秒真实推进一跳；FX 里 UnitID 是起点单位、Slot 是落点单位，
// 电弧整段追着两个球的实时位置走，雷落在身上而不是落在人刚才站的地方。
//
// 渲染完全由 rAF 逐帧驱动（描线进度、明暗、几何全部普通属性/内联数值）：
// 不用 CSS 动画、不用 CSS 变量、不用 innerHTML——追踪每帧都改几何，
// 这类动态交给 CSS 动画在部分引擎会卡住不释放，之前几版看不见线就栽在这。
(function () {
  const SVG_NS = "http://www.w3.org/2000/svg";
  const travelSpeed = 0.9; // 屏幕 px / ms：电弧尖端的爬行速度
  const afterglow = 480; // 描完之后电弧闪烁熄灭的时长 ms

  // 折线形状只在出生时抖一次（沿线参数 t 与垂向系数），之后随端点整体走。
  function buildShape() {
    const shape = [];
    const segs = 6;
    for (let i = 1; i < segs; i++) {
      shape.push([i / segs, (i % 2 ? 1 : -1) * (0.45 + Math.random() * 0.55)]);
    }
    return shape;
  }

  // 单位还活着就跟着它走（style.left/top 是左上角，要加半个宽高才是球心）；
  // 没了（被劈死/离开视野）就用出发时记下的坐标。
  function livePos(el, fx, fy) {
    if (el) {
      const x = parseFloat(el.style.left);
      const y = parseFloat(el.style.top);
      const w = parseFloat(el.style.width);
      const h = parseFloat(el.style.height);
      if (Number.isFinite(x) && Number.isFinite(y)) {
        return [x + (Number.isFinite(w) ? w / 2 : 0), y + (Number.isFinite(h) ? h / 2 : 0)];
      }
    }
    return [fx, fy];
  }

  function impact(x, y, kind) {
    arena.spawnFx("fx-static-flash", x, y, kind);
    for (let i = 0; i < 6; i++) {
      const a = (Math.PI * 2 * i) / 6 + Math.random() * 0.5;
      const r = 14 + Math.random() * 16;
      arena.spawnFx("fx-static-spark", x, y, kind, {
        "--dx": `${Math.cos(a) * r}px`,
        "--dy": `${Math.sin(a) * r}px`,
      });
    }
  }

  arena.registerShot("static-bolt", (fx, ctx) => {
    if (!fx || !ctx || !arena.screenPos || !arena.spawnFx) return;
    const x1f = Number(fx.x);
    const y1f = Number(fx.y);
    const x2f = Number(fx.vx);
    const y2f = Number(fx.vy);
    if (![x1f, y1f, x2f, y2f].every(Number.isFinite)) return;
    const fromEl = document.getElementById(`u-${fx.unitId}`);
    const toEl = document.getElementById(`u-${fx.slot}`);
    const shape = buildShape();
    const len0 = Math.max(1, Math.hypot(x2f - x1f, y2f - y1f));
    const amp = Math.min(15, Math.max(3, len0 * 0.13));
    const travel = Math.min(420, Math.max(180, len0 / travelSpeed));
    const life = travel + afterglow;
    const el = arena.spawnFx("fx-static-bolt", x1f, y1f, fx.kind);
    if (!el) return;
    const svg = document.createElementNS(SVG_NS, "svg");
    const lines = [];
    for (const cls of ["static-bolt-glow", "static-bolt-core"]) {
      const pl = document.createElementNS(SVG_NS, "polyline");
      pl.setAttribute("class", cls);
      pl.setAttribute("fill", "none");
      pl.setAttribute("stroke-linejoin", "round");
      pl.setAttribute("stroke-linecap", "round");
      if (cls === "static-bolt-glow") {
        pl.setAttribute("stroke", "#6fcfff");
        pl.setAttribute("stroke-opacity", "0.55");
        pl.setAttribute("stroke-width", "7");
      } else {
        pl.setAttribute("stroke", "#f2fbff");
        pl.setAttribute("stroke-width", "2.4");
      }
      svg.appendChild(pl);
      lines.push(pl);
    }
    el.appendChild(svg);
    const dmg = Number(fx.amount) || 0;
    const born = performance.now();
    let struck = false;

    function frame(now) {
      if (!el.isConnected) return;
      const t = now - born;
      const [p1x, p1y] = livePos(fromEl, x1f, y1f);
      const [p2x, p2y] = livePos(toEl, x2f, y2f);
      // 屏幕坐标折线 -> 包围盒 -> svg 本地坐标
      const pts = [[p1x, p1y]];
      const dx = p2x - p1x;
      const dy = p2y - p1y;
      const len = Math.hypot(dx, dy);
      const nx = len > 1e-6 ? -dy / len : 0;
      const ny = len > 1e-6 ? dx / len : 1;
      for (const [k, s] of shape) {
        pts.push([p1x + dx * k + nx * amp * s, p1y + dy * k + ny * amp * s]);
      }
      pts.push([p2x, p2y]);
      let minX = Infinity;
      let minY = Infinity;
      let maxX = -Infinity;
      let maxY = -Infinity;
      let total = 0;
      for (let i = 0; i < pts.length; i++) {
        minX = Math.min(minX, pts[i][0]);
        maxX = Math.max(maxX, pts[i][0]);
        minY = Math.min(minY, pts[i][1]);
        maxY = Math.max(maxY, pts[i][1]);
        if (i > 0) total += Math.hypot(pts[i][0] - pts[i - 1][0], pts[i][1] - pts[i - 1][1]);
      }
      const pad = 10;
      minX -= pad;
      minY -= pad;
      maxX += pad;
      maxY += pad;
      const w = Math.max(2, Math.ceil(maxX - minX));
      const h = Math.max(2, Math.ceil(maxY - minY));
      svg.setAttribute("width", String(w));
      svg.setAttribute("height", String(h));
      svg.setAttribute("viewBox", `0 0 ${w} ${h}`);
      const local = pts.map(([px, py]) => `${(px - minX).toFixed(1)},${(py - minY).toFixed(1)}`).join(" ");
      el.style.left = `${((minX + maxX) / 2).toFixed(1)}px`;
      el.style.top = `${((minY + maxY) / 2).toFixed(1)}px`;
      const draw = Math.ceil(total + 2); // 描线总长；dash 全长 = 它
      let p;
      if (t < travel) {
        p = Math.max(0, Math.min(1, t / travel));
        el.style.opacity = "1";
        el.style.filter = "brightness(1.7)";
      } else {
        p = 1;
        const u = Math.min(1, (t - travel) / afterglow);
        el.style.opacity = String((1 - u).toFixed(3));
        el.style.filter = Math.floor(u * 5) % 2 ? "brightness(1.5)" : "brightness(0.9)";
      }
      for (const pl of lines) {
        pl.setAttribute("points", local);
        pl.setAttribute("stroke-dasharray", String(draw));
        pl.setAttribute("stroke-dashoffset", String(Math.round(draw * (1 - p))));
      }
      if (dmg > 0 && !struck && t >= travel) {
        struck = true;
        impact(p2x, p2y, fx.kind); // 尖端够到身体了才炸，且炸在身体此刻的位置
      }
      if (t < life + 60) requestAnimationFrame(frame);
      else el.remove();
    }
    frame(born); // 同步跑第一帧：先藏好再见人，避免整条闪一下
    setTimeout(() => el.remove(), life + 500); // 兜底自清
  });
})();
