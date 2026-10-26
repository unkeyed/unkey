import type { ChartConfig } from "@/components/ui/chart";
import type { ReactNode } from "react";

export type AreaFill = "fade" | "flat";

type FillStyle = {
  gradient: boolean;
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

export const FILL_STYLES: Record<AreaFill, FillStyle> = {
  fade: {
    grid: { stroke: "var(--color-gray-4)", strokeDasharray: "3 3", strokeOpacity: 0.6 },
    cursor: {
      stroke: "var(--color-gray-9)",
      strokeWidth: 1,
      strokeDasharray: "3 3",
      strokeOpacity: 0.5,
    },
    fillOpacity: 1,
    dot: haloDot,
    gradient: true,
  },
  flat: {
    gradient: false,
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
  chartId,
  config,
  fillColors,
}: {
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
          <FadeStops color={fillColors?.[key] ?? config[key].color} />
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
