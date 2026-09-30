import * as React from "react";
import * as RechartsPrimitive from "recharts";
import type { TooltipPayloadEntry } from "recharts";

import { formatCount } from "~/components/analytics/format";
import { cn } from "~/lib/utils";

export type ChartConfig = Record<string, { label?: React.ReactNode; color?: string }>;

const ChartContext = React.createContext<ChartConfig | null>(null);

function useChart(): ChartConfig {
  const config = React.useContext(ChartContext);
  if (!config) {
    throw new Error("useChart must be used within a <ChartContainer />");
  }
  return config;
}

function ChartContainer({
  id,
  className,
  children,
  config,
  ...props
}: React.ComponentProps<"div"> & {
  config: ChartConfig;
  children: React.ComponentProps<typeof RechartsPrimitive.ResponsiveContainer>["children"];
}) {
  const uniqueId = React.useId();
  const chartId = `chart-${id || uniqueId.replace(/:/g, "")}`;

  return (
    <ChartContext.Provider value={config}>
      <div
        data-chart={chartId}
        className={cn(
          "flex aspect-video justify-center text-xs [&_*:focus]:outline-none [&_.recharts-cartesian-axis-tick_text]:fill-muted-foreground [&_.recharts-cartesian-grid_line[stroke='#ccc']]:stroke-border/50 [&_.recharts-curve.recharts-tooltip-cursor]:stroke-border [&_.recharts-dot[stroke='#fff']]:stroke-transparent [&_.recharts-layer]:outline-hidden [&_.recharts-polar-grid_[stroke='#ccc']]:stroke-border [&_.recharts-radial-bar-background-sector]:fill-muted [&_.recharts-rectangle.recharts-tooltip-cursor]:fill-muted [&_.recharts-reference-line_[stroke='#ccc']]:stroke-border [&_.recharts-sector[stroke='#fff']]:stroke-transparent [&_.recharts-sector]:outline-hidden [&_.recharts-surface]:outline-hidden [&_.recharts-wrapper:focus-visible]:rounded-md [&_.recharts-wrapper:focus-visible]:ring-2 [&_.recharts-wrapper:focus-visible]:ring-gray-12",
          className,
        )}
        {...props}
      >
        <ChartStyle id={chartId} config={config} />
        <RechartsPrimitive.ResponsiveContainer initialDimension={{ width: 1, height: 1 }}>
          {children}
        </RechartsPrimitive.ResponsiveContainer>
      </div>
    </ChartContext.Provider>
  );
}

function ChartStyle({ id, config }: { id: string; config: ChartConfig }) {
  const colored = Object.entries(config).filter(([, item]) => item.color);
  if (colored.length === 0) {
    return null;
  }

  return (
    <style
      // biome-ignore lint/security/noDangerouslySetInnerHtml: Dynamic CSS generation for chart theming
      dangerouslySetInnerHTML={{
        __html: `
 [data-chart=${id}] {
${colored.map(([key, item]) => `  --color-${key}: ${item.color};`).join("\n")}
}
`,
      }}
    />
  );
}

const ChartTooltip = RechartsPrimitive.Tooltip;

type TooltipPayload = ReadonlyArray<TooltipPayloadEntry<number, string>>;

type ChartTooltipContentProps = {
  active?: boolean;
  payload?: TooltipPayload;
  labelFormatter: (label: React.ReactNode, payload: TooltipPayload) => React.ReactNode;
  className?: string;
  labelClassName?: string;
};

function ChartTooltipContent({
  active,
  payload,
  labelFormatter,
  className,
  labelClassName,
}: ChartTooltipContentProps) {
  const config = useChart();

  if (!active || !payload?.length) {
    return null;
  }

  return (
    <div
      role="tooltip"
      className={cn(
        "grid select-none items-start gap-1.5 rounded-lg border border-gray-6 bg-background pt-3 pb-2 text-xs shadow-md sm:w-fit md:w-fit md:max-w-[360px]",
        className,
      )}
    >
      <div className={cn("font-medium", labelClassName)}>{labelFormatter(null, payload)}</div>
      <div className="grid gap-0.5">
        {payload.map((item) => {
          const key = String(item.dataKey ?? item.name ?? "value");
          return (
            <div
              key={key}
              className="flex w-full items-center gap-4 px-4 [&>svg]:h-2.5 [&>svg]:w-2.5 [&>svg]:text-muted-foreground"
            >
              <div className="size-2 shrink-0 rounded-xs" style={{ backgroundColor: item.color }} />
              <div className="flex w-full items-center justify-between gap-4 py-1 leading-none">
                <div className="flex items-center gap-4">
                  <span className="text-gray-12 text-xs capitalize">
                    {config[key]?.label ?? item.name}
                  </span>
                </div>
                <div className="ml-auto">
                  {typeof item.value === "number" && (
                    <span className="font-mono text-gray-12 tabular-nums">
                      {formatCount(item.value)}
                    </span>
                  )}
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

export { ChartContainer, ChartTooltip, ChartTooltipContent };
