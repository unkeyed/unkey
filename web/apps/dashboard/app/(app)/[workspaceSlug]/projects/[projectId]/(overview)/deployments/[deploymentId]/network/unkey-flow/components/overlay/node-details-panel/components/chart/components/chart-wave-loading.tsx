import { useId } from "react";

type ChartWaveLoadingProps = {
  height: number;
  color?: string;
};

const WAVE_LINE =
  "M0,70 C30,55 60,80 90,60 C120,40 150,75 180,55 C210,35 240,65 270,50 C300,35 330,60 360,45 C390,30 420,55 450,40 C480,25 510,50 540,35 C570,20 600,45 630,30 C660,15 690,40 720,25";
const WAVE_AREA = `${WAVE_LINE} L720,100 L0,100 Z`;

export function ChartWaveLoading({ height, color }: ChartWaveLoadingProps) {
  const id = useId().replace(/:/g, "");
  const waveColor = color || "var(--gray-8)";

  return (
    <div className="w-full relative animate-pulse" style={{ height }}>
      <svg
        viewBox="0 0 720 100"
        preserveAspectRatio="none"
        className="absolute inset-0 w-full h-full"
      >
        <defs>
          <linearGradient id={`wave-load-${id}`} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor={waveColor} stopOpacity={0.12} />
            <stop offset="100%" stopColor={waveColor} stopOpacity={0.03} />
          </linearGradient>
        </defs>
        <path d={WAVE_AREA} fill={`url(#wave-load-${id})`} />
        <path d={WAVE_LINE} fill="none" stroke={waveColor} strokeWidth="1.5" strokeOpacity={0.25} />
      </svg>
    </div>
  );
}
