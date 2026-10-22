import type { ChartConfig } from "@/components/ui/chart";
import type { ReactNode } from "react";

export type AreaFill = "pulse" | "fade" | "flat";

type FillStyle = {
  gradient: Exclude<AreaFill, "flat"> | null;
  glow: boolean;
  containerClass?: string;
  grid: { stroke: string; strokeDasharray?: string; strokeOpacity: number };
  cursor: {
    stroke?: string;
    strokeWidth: number;
    strokeDasharray?: string;
    strokeOpacity?: number;
  };
  fillOpacity: number;
  dot: (x: number, y: number, color: string | undefined) => ReactNode;
};

function haloDot(x: number, y: number, color: string | undefined) {
  return (
    <g>
      <circle cx={x} cy={y} r={8} fill={color} opacity={0.12} />
      <circle cx={x} cy={y} r={5} fill={color} opacity={0.3} />
      <circle
        cx={x}
        cy={y}
        r={3}
        fill={color}
        stroke="var(--background-color-raised)"
        strokeWidth={1.5}
      />
    </g>
  );
}

const GRADIENT_STYLE = {
  grid: { stroke: "var(--color-gray-4)", strokeDasharray: "3 3", strokeOpacity: 0.6 },
  cursor: {
    stroke: "var(--color-gray-9)",
    strokeWidth: 1,
    strokeDasharray: "3 3",
    strokeOpacity: 0.5,
  },
  fillOpacity: 1,
  dot: haloDot,
};

export const FILL_STYLES: Record<AreaFill, FillStyle> = {
  pulse: { ...GRADIENT_STYLE, gradient: "pulse", glow: true },
  fade: { ...GRADIENT_STYLE, gradient: "fade", glow: false },
  flat: {
    gradient: null,
    glow: false,
    containerClass:
      "[&_.recharts-cartesian-axis-tick_text]:fill-gray-9 [&_.recharts-curve.recharts-tooltip-cursor]:stroke-gray-12",
    grid: { stroke: "var(--color-grayA-3)", strokeOpacity: 1 },
    cursor: { strokeWidth: 1 },
    fillOpacity: 0.08,
    dot: (x, y, color) => (
      <circle
        cx={x}
        cy={y}
        r={3.5}
        fill={color}
        stroke="var(--background-color-raised)"
        strokeWidth={1.5}
      />
    ),
  },
};

export function Gradients({
  kind,
  chartId,
  config,
  fillColors,
}: {
  kind: Exclude<AreaFill, "flat">;
  chartId: string;
  config: ChartConfig;
  fillColors?: Record<string, string>;
}) {
  return (
    <defs>
      {Object.keys(config).map((key) => (
        <linearGradient
          key={`${chartId}-${key}`}
          id={`${chartId}-${key}`}
          x1="0"
          y1="0"
          x2="0"
          y2="1"
        >
          {kind === "fade" ? (
            <FadeStops color={fillColors?.[key] ?? config[key].color} />
          ) : (
            <PulseStops color={config[key].color} />
          )}
        </linearGradient>
      ))}
    </defs>
  );
}

function FadeStops({ color }: { color?: string }) {
  return (
    <>
      <stop offset="0%" stopColor={color} stopOpacity={0.85} />
      <stop offset="90%" stopColor={color} stopOpacity={0.1} />
    </>
  );
}

const PULSE_STOPS = [
  { offset: "0%", opacity: 0.45, values: "0.45;0.3;0.45" },
  { offset: "50%", opacity: 0.2, values: "0.2;0.35;0.2" },
  { offset: "85%", opacity: 0.08, values: "0.08;0.15;0.08" },
];

function PulseStops({ color }: { color?: string }) {
  return (
    <>
      {PULSE_STOPS.map(({ offset, opacity, values }) => (
        <stop key={offset} offset={offset} stopColor={color} stopOpacity={opacity}>
          <animate attributeName="stop-opacity" values={values} dur="6s" repeatCount="indefinite" />
        </stop>
      ))}
      <stop offset="100%" stopColor={color} stopOpacity={0.03} />
    </>
  );
}
