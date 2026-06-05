"use client";

import { cn } from "@unkey/ui/src/lib/utils";
import { type CSSProperties, useEffect, useRef, useState } from "react";

// Unkey "u" favicon geometry, rasterized to an alpha mask at runtime (no shipped image).
const SVG =
  '<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512" viewBox="0 0 512 512">' +
  '<g fill="#000">' +
  '<path d="M170.8 115V340.6H341.2L284.4 397H170.8C139.418 397 114 371.761 114 340.6V115H170.8Z"/>' +
  '<path d="M398 284.2L341.2 340.6V115H398V284.2Z"/>' +
  "</g></svg>";

// Fixed animation/shape params. The size/density bits live in TuneParams so they can be
// experimented with live via the dev panel (append ?asciitune in development).
const C = {
  ca: 0.6, // char aspect nudge (advance ≈ 0.6em for the mono stack)
  lh: 1,
  dot: "▪",
  cutoff: 0.12, // glyph coverage below this is treated as background
  glevel: 1, // dot probability inside the "u"
  grain: 0.06, // random dropout in the "u" for texture
  pow: 3, // cubic falloff
  amb: 0.9, // star peak brightness
  ambSpeed: 0.32, // ambient twinkle speed (slower = calmer, more star-like)
  ambSharp: 4, // higher = briefer, sparklier star blinks
  rev: 1100, // mount dither-in duration (ms)
};
const SRC = 256;
const SS = 2;
const DIRS = [
  [-1, 0],
  [0, 1],
  [1, 0],
  [0, -1],
] as const; // up, right, down, left (snake)
const DIRS8 = [
  [-1, 0],
  [1, 0],
  [0, -1],
  [0, 1],
  [-1, -1],
  [-1, 1],
  [1, -1],
  [1, 1],
] as const; // 8-way (burst)

type Flow = "off" | "rain" | "burst";
type Motion = "none" | "snake" | "rain" | "burst";

type TuneParams = {
  cols: number; // grid resolution — more columns = finer, denser dots
  dotSize: number; // font size in px (dot size)
  gap: number; // px between dots
  flevel: number; // background scatter density
  starDensity: number; // fraction of background dots that twinkle
  gridMode: boolean; // background dots on a uniform lattice instead of random scatter
  gridStride: number; // lattice spacing in cells when gridMode is on
  bgOn: boolean; // show background dots at all
  solidGlyph: boolean; // force every "u" cell on so the mark never has gaps
  edgeFade: number; // 0 = hard rectangular field, 1 = strong vignette toward the edges
  radius: number; // cursor influence radius (px)
  boost: number; // density added under the cursor
  rise: number; // how fast cells light up (lower = more ambient)
  decay: number; // trailing fade after the cursor leaves (higher = longer trail)
  twrate: number; // digit-flip cadence in binary data-stream mode
  binary: boolean; // render 0/1 digits (brand style) instead of square dots
  digitFlip: boolean; // when binary, animated cells flip 0<->1 like a data stream
  snake: boolean; // background crawls as snake trails instead of twinkling
  snakeCount: number; // number of trails/streams
  snakeSpeed: number; // steps per second
  snakeLen: number; // trail length in cells
  flow: Flow; // directional stream engine: "rain" falls (matrix), "burst" radiates (goku)
  headColor: string; // bright head/glyph color ("" = theme); e.g. matrix green, goku gold
  tailColor: string; // dim trail/field color ("" = theme)
  tintGlyph: boolean; // tint the "u" with headColor (vs leaving it the neutral theme color)
};

// keys that change per-cell geometry/placement/colour and need a rebuild; the rest are live.
const STRUCTURAL: ReadonlySet<keyof TuneParams> = new Set([
  "cols",
  "dotSize",
  "gap",
  "flevel",
  "starDensity",
  "gridMode",
  "gridStride",
  "bgOn",
  "solidGlyph",
  "edgeFade",
  "snake",
  "snakeCount",
  "flow",
  "headColor",
  "tailColor",
  "tintGlyph",
]);

const TUNE_KEY = "unkey-ascii-tune"; // dev-only: persists panel tweaks across refreshes

// default look (= the daves-v1 preset)
const DEFAULTS: TuneParams = {
  cols: 50,
  dotSize: 3,
  gap: 1.25,
  flevel: 0.1,
  starDensity: 0.33,
  gridMode: false,
  gridStride: 1,
  bgOn: true,
  solidGlyph: true,
  edgeFade: 0.75,
  radius: 56,
  boost: 1.2,
  rise: 0.28,
  decay: 0.88,
  twrate: 0.05,
  binary: true,
  digitFlip: false,
  snake: false,
  snakeCount: 16,
  snakeSpeed: 5,
  snakeLen: 9,
  flow: "off",
  headColor: "",
  tailColor: "",
  tintGlyph: true,
};

const PRESETS: { name: string; params: TuneParams }[] = [
  { name: "daves-v1", params: { ...DEFAULTS } },
  {
    name: "matrix-mode",
    params: {
      ...DEFAULTS,
      cols: 56,
      dotSize: 3,
      gap: 1.25,
      flevel: 0.04,
      starDensity: 0,
      edgeFade: 0.6,
      boost: 0.16,
      binary: true,
      snakeCount: 26,
      snakeSpeed: 11,
      snakeLen: 11,
      flow: "rain",
      headColor: "#caffda",
      tailColor: "#34c06a",
      tintGlyph: true,
    },
  },
  {
    name: "goku-supersaiyan",
    params: {
      ...DEFAULTS,
      cols: 55,
      dotSize: 3,
      gap: 1.25,
      flevel: 0.03,
      starDensity: 0,
      edgeFade: 0.5,
      boost: 0.2,
      binary: false,
      snakeCount: 30,
      snakeSpeed: 13,
      snakeLen: 6,
      flow: "burst",
      headColor: "#fff4bf",
      tailColor: "#ff9b1a",
      tintGlyph: true,
    },
  },
];

export function UnkeyAscii({
  fontSize = DEFAULTS.dotSize,
  className,
}: {
  fontSize?: number;
  className?: string;
}) {
  const fieldRef = useRef<HTMLPreElement>(null); // dim trails / background field (base, pointer)
  const brightRef = useRef<HTMLPreElement>(null); // bright stream heads
  const logoRef = useRef<HTMLPreElement>(null); // the "u"
  const params = useRef<TuneParams>({ ...DEFAULTS, dotSize: fontSize });
  const rebuildRef = useRef<(reveal: boolean) => void>(() => {});

  const [tune, setTune] = useState<TuneParams | null>(null); // non-null => dev panel visible

  useEffect(() => {
    if (process.env.NODE_ENV === "production") {
      return;
    }
    // Persisted panel tweaks apply only while actively tuning (?asciitune). Otherwise the page
    // always renders the committed default (daves-v1) so saved experiments don't leak into it.
    if (!new URLSearchParams(window.location.search).has("asciitune")) {
      return;
    }
    try {
      const saved = localStorage.getItem(TUNE_KEY);
      if (saved) {
        params.current = { ...params.current, ...(JSON.parse(saved) as Partial<TuneParams>) };
      }
    } catch {
      // ignore unreadable/corrupt persisted tweaks
    }
    setTune({ ...params.current });
  }, []);

  useEffect(() => {
    const fieldEl = fieldRef.current;
    const brightEl = brightRef.current;
    const logoEl = logoRef.current;
    const measEl = document.createElement("canvas").getContext("2d");
    const mc = document.createElement("canvas");
    const maskEl = mc.getContext("2d", { willReadFrequently: true });
    if (!fieldEl || !brightEl || !logoEl || !measEl || !maskEl) {
      return;
    }
    const field = fieldEl;
    const bright = brightEl;
    const logo = logoEl;
    const measCtx = measEl;
    const maskCtx = maskEl;

    const reduced = matchMedia("(prefers-reduced-motion: reduce)").matches;
    let raf = 0;
    let running = false;
    let revStart = -1e9;
    let cols = 0;
    let rows = 0;
    let wc = 0;
    let hc = 0;
    let centerR = 0;
    let centerC = 0;
    let motion: Motion = "none";
    let base = new Float32Array(0);
    let energy = new Float32Array(0);
    let thr = new Float32Array(0);
    let rev = new Float32Array(0);
    let phase = new Float32Array(0);
    let isField = new Uint8Array(0);
    let isStar = new Uint8Array(0);
    let bit = new Uint8Array(0); // per-cell 0/1 for binary mode
    let fade = new Float32Array(0); // per-cell edge vignette (1 = full, 0 = faded out)
    let occ = new Uint8Array(0); // cells covered by a stream trail
    let occHead = new Uint8Array(0); // cells that are a stream head (bright)
    let streams: { r: number; c: number; dr: number; dc: number; body: number[] }[] = [];
    let streamAcc = 0; // accumulated ms toward the next step
    let streamLast = 0; // timestamp of previous frame
    let mask: Float32Array | null = null;
    const ptr = { x: -1e9, y: -1e9, inside: false };

    const ensure = () => {
      if (!running) {
        running = true;
        raf = requestAnimationFrame(loop);
      }
    };

    function spawnStream(initial: boolean) {
      if (motion === "rain") {
        const c0 = (Math.random() * cols) | 0;
        const r0 = initial ? (Math.random() * rows) | 0 : 0;
        return { r: r0, c: c0, dr: 1, dc: 0, body: [r0 * cols + c0] };
      }
      if (motion === "burst") {
        const d = DIRS8[(Math.random() * DIRS8.length) | 0];
        return { r: centerR, c: centerC, dr: d[0], dc: d[1], body: [centerR * cols + centerC] };
      }
      const r0 = (Math.random() * rows) | 0;
      const c0 = (Math.random() * cols) | 0;
      const d = DIRS[(Math.random() * 4) | 0];
      return { r: r0, c: c0, dr: d[0], dc: d[1], body: [r0 * cols + c0] };
    }

    function advance(s: (typeof streams)[number]) {
      if (motion === "snake") {
        if (Math.random() < 0.12) {
          const ndr = Math.random() < 0.5 ? s.dc : -s.dc;
          const ndc = Math.random() < 0.5 ? -s.dr : s.dr;
          s.dr = ndr;
          s.dc = ndc;
        }
        s.r = (s.r + s.dr + rows) % rows;
        s.c = (s.c + s.dc + cols) % cols;
        s.body.unshift(s.r * cols + s.c);
        return;
      }
      const nr = s.r + s.dr;
      const nc = s.c + s.dc;
      if (nr < 0 || nr >= rows || nc < 0 || nc >= cols) {
        const ns = spawnStream(false); // walked off-grid — respawn (top for rain, centre for burst)
        s.r = ns.r;
        s.c = ns.c;
        s.dr = ns.dr;
        s.dc = ns.dc;
        s.body.length = 0;
        s.body.push(s.r * cols + s.c);
        return;
      }
      s.r = nr;
      s.c = nc;
      s.body.unshift(s.r * cols + s.c);
    }

    function build(reveal: boolean) {
      if (!mask) {
        return;
      }
      const maskData = mask;
      const p = params.current;
      measCtx.font = `${p.dotSize}px ui-monospace, monospace`;
      wc = measCtx.measureText("0").width * (C.ca / 0.6) + p.gap;
      hc = p.dotSize * C.lh + p.gap;
      cols = p.cols;
      rows = Math.max(4, Math.round((cols * wc) / hc));
      centerR = (rows / 2) | 0;
      centerC = (cols / 2) | 0;
      motion = p.flow !== "off" ? p.flow : p.snake ? "snake" : "none";
      for (const el of [field, bright, logo]) {
        el.style.fontSize = `${p.dotSize}px`;
        el.style.lineHeight = `${hc}px`;
        el.style.letterSpacing = `${p.gap}px`;
      }
      // colours: "" falls back to the Tailwind class (theme-aware)
      logo.style.color = p.tintGlyph && p.headColor ? p.headColor : "";
      bright.style.color = p.headColor || "";
      field.style.color = p.tailColor || "";

      const n = cols * rows;
      base = new Float32Array(n);
      energy = new Float32Array(n);
      thr = new Float32Array(n);
      rev = new Float32Array(n);
      phase = new Float32Array(n);
      isField = new Uint8Array(n);
      isStar = new Uint8Array(n);
      bit = new Uint8Array(n);
      fade = new Float32Array(n);
      occ = new Uint8Array(n);
      occHead = new Uint8Array(n);
      streams = [];
      if (motion !== "none") {
        const sc = Math.max(0, Math.round(p.snakeCount));
        for (let k = 0; k < sc; k++) {
          streams.push(spawnStream(true));
        }
      }
      streamAcc = 0;
      const cx = (cols - 1) / 2;
      const cy = (rows - 1) / 2;
      const maxR = Math.hypot(cx, cy);
      const at = (u: number, v: number) => {
        const x = Math.min(SRC - 1, Math.max(0, (u * SRC) | 0));
        const y = Math.min(SRC - 1, Math.max(0, (v * SRC) | 0));
        return maskData[y * SRC + x];
      };
      for (let r = 0; r < rows; r++) {
        for (let c = 0; c < cols; c++) {
          const i = r * cols + c;
          let acc = 0;
          for (let sy = 0; sy < SS; sy++) {
            for (let sx = 0; sx < SS; sx++) {
              acc += at((c + (sx + 0.5) / SS) / cols, (r + (sy + 0.5) / SS) / rows);
            }
          }
          const cov = acc / (SS * SS);
          // Solid fill uses 50%-coverage membership so the mark keeps its true weight; the
          // dithered mode keeps the low cutoff for soft antialiased edges.
          const bg = p.solidGlyph ? cov < 0.5 : cov <= C.cutoff;
          // soft elliptical vignette so the background fades out toward the edges (corners first)
          // instead of ending on a hard rectangle. Only applied to the background, never the "u".
          const nx = (c - cx) / (cols / 2);
          const ny = (r - cy) / (rows / 2);
          const radial = Math.hypot(nx, ny);
          const innerR = 1.42 * (1 - p.edgeFade);
          let vig = 1;
          if (p.edgeFade > 0 && radial > innerR) {
            const tt = Math.min(1, (radial - innerR) / (1.42 - innerR));
            vig = 1 - tt * tt * (3 - 2 * tt);
          }
          fade[i] = bg ? vig : 1;
          let b: number;
          if (!bg) {
            b = p.solidGlyph
              ? 1 // every cell of the "u" stays on — no gaps from grain/dither
              : p.flevel + Math.min(1, (cov - C.cutoff) / 0.18) * (C.glevel - p.flevel) - C.grain * Math.random();
          } else if (!p.bgOn) {
            b = 0;
          } else if (p.gridMode) {
            // background dots on a uniform lattice
            b = (r % p.gridStride === 0 && c % p.gridStride === 0 ? 1 : 0) * vig;
          } else {
            b = p.flevel * vig; // random scatter (thr decides which cells show)
          }
          base[i] = Math.max(0, b);
          isField[i] = bg ? 1 : 0;
          isStar[i] = bg && p.bgOn && Math.random() < p.starDensity ? 1 : 0;
          bit[i] = Math.random() < 0.5 ? 1 : 0;
          thr[i] = Math.random();
          phase[i] = Math.random() * Math.PI * 2;
          // first-load order: rain cascades in top-to-bottom (Matrix), everything else
          // grows out from the centre (a burst for goku, a gentle dither otherwise).
          rev[i] =
            motion === "rain"
              ? (r / Math.max(1, rows - 1)) * 0.8 + 0.2 * Math.random()
              : 0.65 * (Math.hypot(c - cx, r - cy) / maxR) + 0.35 * Math.random();
        }
      }
      revStart = reveal ? performance.now() : -1e9;
      ensure();
    }
    rebuildRef.current = build;

    function loop(now: number) {
      const p = params.current;
      const revP = reduced || C.rev === 0 ? 1 : Math.min(1, (now - revStart) / C.rev);
      let active = revP < 1 || ptr.inside || (!reduced && (C.amb > 0 || motion !== "none"));
      let fieldStr = "";
      let brightStr = "";
      let logoStr = "";

      if (!reduced && motion !== "none") {
        // advance the streams on a fixed step cadence, independent of frame rate
        const interval = 1000 / Math.max(0.5, p.snakeSpeed);
        streamAcc += Math.min(120, now - streamLast);
        let guard = 0;
        while (streamAcc >= interval && guard < 8) {
          streamAcc -= interval;
          guard++;
          for (const s of streams) {
            advance(s);
          }
        }
        const len = Math.max(1, Math.round(p.snakeLen));
        occ.fill(0);
        occHead.fill(0);
        for (const s of streams) {
          while (s.body.length > len) {
            s.body.pop();
          }
          for (let k = 0; k < s.body.length; k++) {
            occ[s.body[k]] = 1;
          }
          if (s.body.length > 0) {
            occHead[s.body[0]] = 1;
          }
        }
      }
      streamLast = now;

      const headBright = motion === "rain" || motion === "burst";
      for (let r = 0; r < rows; r++) {
        for (let c = 0; c < cols; c++) {
          const i = r * cols + c;
          let e = energy[i];
          if (!reduced) {
            let target = 0;
            if (ptr.inside) {
              const d = Math.hypot((c + 0.5) * wc - ptr.x, (r + 0.5) * hc - ptr.y);
              if (d < p.radius) {
                target = (1 - d / p.radius) ** C.pow;
              }
            }
            e = target > e ? e + (target - e) * p.rise : e * p.decay;
            if (e < 0.004) {
              e = 0;
            }
            energy[i] = e;
            if (e > 0) {
              active = true;
            }
          }
          const rv = revP >= 1 ? 1 : Math.max(0, Math.min(1, (revP - rev[i]) / 0.28));
          // hover adds density only: energy raises the probability against a STATIC threshold,
          // so cells steadily fill in near the cursor and fade with the trail — no flicker.
          let prob = (base[i] + e * p.boost) * rv;
          const t = thr[i];
          let animating = false;
          if (!reduced && motion === "none" && isStar[i] && C.amb > 0) {
            // a sparse few background dots slowly pulse in and out of phase, like stars.
            animating = true;
            const ph = phase[i];
            const sp = C.ambSpeed * (0.7 + 0.6 * (ph / 6.2832));
            const tw = 0.5 + 0.5 * Math.sin(now * 0.001 * sp + ph);
            prob = Math.max(prob, C.amb * tw ** C.ambSharp * rv * fade[i]);
          }
          // stream trails (snake / rain / burst), dithered by the vignette so they fade at edges
          const flowOn =
            motion !== "none" && isField[i] === 1 && occ[i] === 1 && fade[i] * rv > thr[i];
          if (flowOn) {
            animating = true;
          }
          const on = prob > t || flowOn;

          let ch = C.dot;
          if (p.binary) {
            if (p.digitFlip && animating && Math.random() < p.twrate) {
              bit[i] ^= 1; // data-stream flicker on active cells
            }
            ch = bit[i] ? "1" : "0";
          }

          // route each lit cell to one of three layers: the "u", bright stream heads, dim rest
          let g = " ";
          let h = " ";
          let dchar = " ";
          if (on) {
            if (isField[i] === 0) {
              g = ch;
            } else if (headBright && flowOn && occHead[i] === 1) {
              h = ch;
            } else {
              dchar = ch;
            }
          }
          logoStr += g;
          brightStr += h;
          fieldStr += dchar;
        }
        if (r < rows - 1) {
          fieldStr += "\n";
          brightStr += "\n";
          logoStr += "\n";
        }
      }
      field.textContent = fieldStr;
      bright.textContent = brightStr;
      logo.textContent = logoStr;
      if (active) {
        raf = requestAnimationFrame(loop);
      } else {
        running = false;
      }
    }

    const onMove = (e: PointerEvent) => {
      const b = field.getBoundingClientRect();
      ptr.x = e.clientX - b.left;
      ptr.y = e.clientY - b.top;
      ptr.inside = true;
      ensure();
    };
    const onLeave = () => {
      ptr.inside = false;
      ensure();
    };
    field.addEventListener("pointermove", onMove);
    field.addEventListener("pointerleave", onLeave);

    const img = new Image();
    img.onload = () => {
      mc.width = SRC;
      mc.height = SRC;
      maskCtx.clearRect(0, 0, SRC, SRC);
      maskCtx.drawImage(img, 0, 0, SRC, SRC);
      const a = maskCtx.getImageData(0, 0, SRC, SRC).data;
      mask = new Float32Array(SRC * SRC);
      for (let i = 0; i < mask.length; i++) {
        mask[i] = a[i * 4 + 3] / 255;
      }
      build(true);
    };
    img.src = `data:image/svg+xml,${encodeURIComponent(SVG)}`;

    return () => {
      cancelAnimationFrame(raf);
      field.removeEventListener("pointermove", onMove);
      field.removeEventListener("pointerleave", onLeave);
    };
  }, []);

  const preStyle: CSSProperties = {
    margin: 0,
    whiteSpace: "pre",
    userSelect: "none",
    fontFamily: "var(--font-mono, ui-monospace, monospace)",
  };
  const overlayStyle: CSSProperties = { ...preStyle, pointerEvents: "none" };

  // Apply a live edit from the dev panel. Geometry/colour changes rebuild (no reveal replay);
  // hover/animation changes are picked up by the running loop without a rebuild.
  const edit = (patch: Partial<TuneParams>) => {
    params.current = { ...params.current, ...patch };
    setTune({ ...params.current });
    try {
      localStorage.setItem(TUNE_KEY, JSON.stringify(params.current));
    } catch {
      // ignore quota/availability errors
    }
    if (Object.keys(patch).some((k) => STRUCTURAL.has(k as keyof TuneParams))) {
      rebuildRef.current(false);
    }
  };

  const reset = () => {
    params.current = { ...DEFAULTS, dotSize: fontSize };
    setTune({ ...params.current });
    try {
      localStorage.removeItem(TUNE_KEY);
    } catch {
      // ignore availability errors
    }
    rebuildRef.current(false);
  };

  return (
    <div className={cn("relative inline-block", className)} aria-hidden>
      <pre ref={fieldRef} className="text-gray-8" style={preStyle} />
      <pre ref={brightRef} className="absolute inset-0 text-accent-12" style={overlayStyle} />
      <pre ref={logoRef} className="absolute inset-0 text-accent-12" style={overlayStyle} />
      {tune ? <TunePanel tune={tune} edit={edit} reset={reset} /> : null}
    </div>
  );
}

function TunePanel({
  tune,
  edit,
  reset,
}: {
  tune: TuneParams;
  edit: (patch: Partial<TuneParams>) => void;
  reset: () => void;
}) {
  const row = (label: string, key: keyof TuneParams, min: number, max: number, step: number) => (
    <label style={{ display: "flex", flexDirection: "column", gap: 2 }}>
      <span style={{ display: "flex", justifyContent: "space-between" }}>
        <span>{label}</span>
        <span>{String(tune[key])}</span>
      </span>
      <input
        type="range"
        min={min}
        max={max}
        step={step}
        value={Number(tune[key])}
        onChange={(e) => edit({ [key]: Number.parseFloat(e.target.value) })}
      />
    </label>
  );
  const check = (label: string, key: keyof TuneParams) => (
    <label style={{ display: "flex", gap: 6, alignItems: "center" }}>
      <input
        type="checkbox"
        checked={Boolean(tune[key])}
        onChange={(e) => edit({ [key]: e.target.checked })}
      />
      {label}
    </label>
  );
  const colorRow = (label: string, key: "headColor" | "tailColor") => (
    <label style={{ display: "flex", flexDirection: "column", gap: 2 }}>
      <span>{label}</span>
      <input
        type="text"
        value={tune[key]}
        placeholder="theme"
        onChange={(e) => edit({ [key]: e.target.value })}
        style={{
          background: "#0b0b0b",
          border: "1px solid #333",
          color: "#ededed",
          borderRadius: 6,
          padding: "4px 6px",
          font: "11px ui-monospace, monospace",
        }}
      />
    </label>
  );

  const trails = tune.snake || tune.flow !== "off";

  return (
    <div
      style={{
        position: "fixed",
        bottom: 16,
        right: 16,
        zIndex: 9999,
        width: 240,
        maxHeight: "92vh",
        overflowY: "auto",
        display: "flex",
        flexDirection: "column",
        gap: 8,
        padding: 12,
        borderRadius: 8,
        background: "rgba(10,10,10,0.92)",
        border: "1px solid #333",
        color: "#ededed",
        font: "11px ui-monospace, monospace",
      }}
    >
      <strong>ASCII tuning (dev)</strong>
      {row("Columns", "cols", 24, 140, 1)}
      {row("Dot size (px)", "dotSize", 1, 14, 0.25)}
      {row("Dot spacing (px)", "gap", 0, 6, 0.25)}
      {row("Field density", "flevel", 0, 0.5, 0.005)}
      {row("Star density", "starDensity", 0, 0.4, 0.01)}
      {row("Edge fade", "edgeFade", 0, 1, 0.05)}
      {check("Background dots on", "bgOn")}
      {check('Solid "u" (no gaps)', "solidGlyph")}
      {check("Grid mode (uniform lattice)", "gridMode")}
      {tune.gridMode ? row("Grid stride", "gridStride", 1, 6, 1) : null}
      {check("Binary (0/1) digits", "binary")}
      {tune.binary ? check("Flip digits (data stream)", "digitFlip") : null}
      {tune.binary && tune.digitFlip ? row("Flip rate", "twrate", 0.05, 1, 0.05) : null}

      <strong style={{ marginTop: 4 }}>Background motion</strong>
      {check("Snake (wander)", "snake")}
      <label style={{ display: "flex", gap: 6, alignItems: "center" }}>
        <input
          type="checkbox"
          checked={tune.flow === "rain"}
          onChange={(e) => edit({ flow: e.target.checked ? "rain" : "off" })}
        />
        Rain flow (matrix)
      </label>
      <label style={{ display: "flex", gap: 6, alignItems: "center" }}>
        <input
          type="checkbox"
          checked={tune.flow === "burst"}
          onChange={(e) => edit({ flow: e.target.checked ? "burst" : "off" })}
        />
        Burst flow (goku)
      </label>
      {trails ? (
        <>
          {row("Trail count", "snakeCount", 1, 40, 1)}
          {row("Trail speed", "snakeSpeed", 1, 30, 1)}
          {row("Trail length", "snakeLen", 2, 24, 1)}
        </>
      ) : null}

      <strong style={{ marginTop: 4 }}>Color</strong>
      {check('Tint "u"', "tintGlyph")}
      {colorRow("Head / bright", "headColor")}
      {colorRow("Tail / dim", "tailColor")}

      <strong style={{ marginTop: 4 }}>Hover (density)</strong>
      {row("Radius (px)", "radius", 20, 220, 2)}
      {row("Boost", "boost", 0, 1.2, 0.02)}
      {row("Rise speed", "rise", 0.05, 1, 0.01)}
      {row("Trail decay", "decay", 0.7, 0.99, 0.005)}

      <div style={{ display: "flex", gap: 6 }}>
        <button
          type="button"
          style={{ flex: 1, padding: "6px 8px", cursor: "pointer", borderRadius: 6, border: "1px solid #333", background: "#ededed", color: "#000" }}
          onClick={() => navigator.clipboard?.writeText(JSON.stringify(tune, null, 2))}
        >
          Copy config
        </button>
        <button
          type="button"
          style={{ padding: "6px 8px", cursor: "pointer", borderRadius: 6, border: "1px solid #333", background: "transparent", color: "#ededed" }}
          onClick={reset}
        >
          Reset
        </button>
      </div>
      <strong style={{ marginTop: 4 }}>Presets</strong>
      {PRESETS.map((pr) => (
        <button
          key={pr.name}
          type="button"
          style={{ padding: "6px 8px", cursor: "pointer", borderRadius: 6, border: "1px solid #333", background: "transparent", color: "#ededed", textAlign: "left" }}
          onClick={() => edit(pr.params)}
        >
          {pr.name}
        </button>
      ))}
    </div>
  );
}
