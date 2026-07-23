import { cn } from "@/lib/utils";

export type Mark = "line" | "bars" | "ratio" | "heatmap";

const ORANGE = "hsl(16 90% 52%)";

function lastN(points: number[], n: number): number[] {
  return points.length > n ? points.slice(points.length - n) : points;
}

export function RowMark({
  mark,
  points,
  errorRatio,
  stroke,
  className,
}: {
  mark: Mark;
  points: number[];
  errorRatio: number;
  stroke: string;
  className?: string;
}) {
  if (points.length === 0 && mark !== "ratio") {
    return <div className={cn("rounded bg-grayA-3", className)} />;
  }
  switch (mark) {
    case "bars":
      return <BarsMark points={points} errorRatio={errorRatio} stroke={stroke} />;
    case "ratio":
      return <RatioMark errorRatio={errorRatio} stroke={stroke} className={className} />;
    case "heatmap":
      return <HeatmapMark points={points} stroke={stroke} className={className} />;
    default:
      return <LineMark points={points} stroke={stroke} className={className} />;
  }
}

// Full-width background chart for the hybrid row: a smooth line with a gradient
// area that fades upward, so text above it stays readable.
export function HybridChart({
  points,
  stroke,
  className,
}: {
  points: number[];
  stroke: string;
  className?: string;
}) {
  if (points.length === 0) {
    return null;
  }
  const w = 800;
  const h = 100;
  const pad = 10;
  const max = Math.max(...points);
  const min = Math.min(...points);
  const range = max - min || 1;
  const step = points.length > 1 ? w / (points.length - 1) : w;
  const coords = points.map((p, i) => {
    const x = i * step;
    const y = h - pad - ((p - min) / range) * (h - pad * 2);
    return [x, y] as const;
  });
  let d = `M${coords[0][0]},${coords[0][1]}`;
  for (let i = 1; i < coords.length; i++) {
    const [px, py] = coords[i - 1];
    const [x, y] = coords[i];
    d += ` Q${px},${py} ${(px + x) / 2},${(py + y) / 2}`;
  }
  d += ` L${coords[coords.length - 1][0]},${coords[coords.length - 1][1]}`;
  const area = `${d} L${w},${h} L0,${h} Z`;
  const gid = `hybgrad-${stroke.replace(/[^a-z0-9]/gi, "")}`;
  return (
    <svg viewBox={`0 0 ${w} ${h}`} className={className} preserveAspectRatio="none" aria-hidden>
      <defs>
        <linearGradient id={gid} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor={stroke} stopOpacity="0" />
          <stop offset="100%" stopColor={stroke} stopOpacity="0.22" />
        </linearGradient>
      </defs>
      <path d={area} fill={`url(#${gid})`} />
      <path
        d={d}
        fill="none"
        stroke={stroke}
        strokeOpacity="0.5"
        strokeWidth="1.5"
        vectorEffect="non-scaling-stroke"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

function LineMark({
  points,
  stroke,
  className,
}: {
  points: number[];
  stroke: string;
  className?: string;
}) {
  const w = 120;
  const h = 40;
  const padY = 7;
  const max = Math.max(...points);
  const min = Math.min(...points);
  const range = max - min || 1;
  const step = points.length > 1 ? w / (points.length - 1) : w;
  const coords = points.map((p, i) => {
    const x = i * step;
    const y = h - padY - ((p - min) / range) * (h - padY * 2);
    return [x, y] as const;
  });
  let d = `M${coords[0][0]},${coords[0][1]}`;
  for (let i = 1; i < coords.length; i++) {
    const [px, py] = coords[i - 1];
    const [x, y] = coords[i];
    const mx = (px + x) / 2;
    const my = (py + y) / 2;
    d += ` Q${px},${py} ${mx},${my}`;
  }
  const [lastX, lastY] = coords[coords.length - 1];
  d += ` L${lastX},${lastY}`;
  const area = `${d} L${w},${h} L0,${h} Z`;
  return (
    <svg viewBox={`0 0 ${w} ${h}`} className={className} preserveAspectRatio="none" aria-hidden>
      <path d={area} fill={stroke} fillOpacity={0.12} />
      <path
        d={d}
        fill="none"
        stroke={stroke}
        strokeWidth={2}
        vectorEffect="non-scaling-stroke"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <circle cx={lastX} cy={lastY} r={2} fill={stroke} vectorEffect="non-scaling-stroke" />
    </svg>
  );
}

// Mirrors the portal key-usage sparkline: thin fixed-width valid bars (tinted per
// resource kind) with a red error cap on top, in a subtle rounded well.
function BarsMark({
  points,
  errorRatio,
  stroke,
}: {
  points: number[];
  errorRatio: number;
  stroke: string;
}) {
  const data = lastN(points, 30);
  const H = 28;
  const max = Math.max(...data, 1) * 1.3;
  return (
    <div className="inline-flex h-7 items-end gap-[2px] rounded-sm bg-gray-2 px-1 transition-colors group-hover:bg-gray-3">
      {data.map((v, i) => {
        const total = Math.min(Math.round((v / max) * H), H);
        const top = errorRatio > 0 && v > 0 ? Math.max(Math.round(errorRatio * total), 1) : 0;
        const bottom = Math.max(total - top, 0);
        return (
          // biome-ignore lint/suspicious/noArrayIndexKey: positional bars
          <div key={i} className="flex w-[3px] shrink-0 flex-col justify-end">
            <div className="w-full bg-error-9" style={{ height: `${top}px` }} />
            <div className="w-full" style={{ height: `${bottom}px`, backgroundColor: stroke }} />
          </div>
        );
      })}
    </div>
  );
}

function RatioMark({
  errorRatio,
  stroke,
  className,
}: {
  errorRatio: number;
  stroke: string;
  className?: string;
}) {
  const validPct = Math.max(0, Math.min(100, (1 - errorRatio) * 100));
  return (
    <div className={cn("flex items-center", className)}>
      <div className="w-full h-2 rounded-full overflow-hidden flex bg-grayA-3">
        <div style={{ width: `${validPct}%`, backgroundColor: stroke }} className="h-full" />
        <div style={{ width: `${100 - validPct}%`, backgroundColor: ORANGE }} className="h-full" />
      </div>
    </div>
  );
}

function HeatmapMark({
  points,
  stroke,
  className,
}: {
  points: number[];
  stroke: string;
  className?: string;
}) {
  const data = lastN(points, 12);
  const max = Math.max(...data, 1);
  return (
    <div className={cn("flex items-center gap-[2px]", className)}>
      {data.map((v, i) => (
        <div
          // biome-ignore lint/suspicious/noArrayIndexKey: positional cells
          key={i}
          className="h-4 flex-1 rounded-[2px]"
          style={{ backgroundColor: stroke, opacity: 0.15 + 0.85 * (v / max) }}
        />
      ))}
    </div>
  );
}
