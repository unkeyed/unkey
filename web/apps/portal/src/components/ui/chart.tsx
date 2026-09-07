import * as React from "react";
import * as RechartsPrimitive from "recharts";

import { cn } from "~/lib/utils";

const numberFormat = new Intl.NumberFormat("en-US");

function formatNumber(value: number): string {
  return numberFormat.format(value);
}

const THEMES = { light: "", dark: ".dark" } as const;

export type ChartConfig = {
  [k in string]: {
    label?: React.ReactNode;
    subLabel?: React.ReactNode;
    icon?: React.ComponentType;
  } & (
    | { color?: string; theme?: never }
    | { color?: never; theme: Record<keyof typeof THEMES, string> }
  );
};

type ChartContextProps = {
  config: ChartConfig;
};

const ChartContext = React.createContext<ChartContextProps | null>(null);

function useChart() {
  const context = React.useContext(ChartContext);

  if (!context) {
    throw new Error("useChart must be used within a <ChartContainer />");
  }

  return context;
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
    <ChartContext.Provider value={{ config }}>
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
  const colorConfig = Object.entries(config).filter(([_, config]) => config.theme || config.color);

  if (!colorConfig.length) {
    return null;
  }

  return (
    <style
      // biome-ignore lint/security/noDangerouslySetInnerHtml: Dynamic CSS generation for chart theming
      dangerouslySetInnerHTML={{
        __html: Object.entries(THEMES)
          .map(
            ([theme, prefix]) => `
${prefix} [data-chart=${id}] {
${colorConfig
  .map(([key, itemConfig]) => {
    const color = itemConfig.theme?.[theme as keyof typeof itemConfig.theme] || itemConfig.color;
    return color ? `  --color-${key}: ${color};` : null;
  })
  .join("\n")}
}
`,
          )
          .join("\n"),
      }}
    />
  );
}

const ChartTooltip = RechartsPrimitive.Tooltip;

function ChartTooltipContent({
  active,
  payload: rawPayload,
  className,
  indicator = "dot",
  hideLabel = false,
  hideIndicator = false,
  label,
  labelFormatter,
  labelClassName,
  color,
  nameKey,
  labelKey,
  bottomExplainer,
  ref,
}: React.ComponentProps<typeof RechartsPrimitive.Tooltip> &
  React.ComponentProps<"div"> & {
    hideLabel?: boolean;
    hideIndicator?: boolean;
    indicator?: "line" | "dot" | "dashed";
    nameKey?: string;
    labelKey?: string;
    bottomExplainer?: React.ReactNode;
    payload?: unknown;
    label?: unknown;
  }) {
  const { config } = useChart();

  // Type guard for payload
  const payload = Array.isArray(rawPayload) ? rawPayload : [];

  const tooltipLabel = React.useMemo(() => {
    if (hideLabel || !payload.length) {
      return null;
    }

    const [item] = payload;
    const key = `${labelKey || item?.dataKey || item?.name || "value"}`;
    const itemConfig = getPayloadConfigFromPayload(config, item, key);
    const value =
      !labelKey && typeof label === "string" ? config[label]?.label || label : itemConfig?.label;

    if (labelFormatter) {
      return (
        <div className={cn("font-medium", labelClassName)}>{labelFormatter(value, payload)}</div>
      );
    }

    if (!value) {
      return null;
    }

    return <div className={cn("font-medium", labelClassName)}>{value}</div>;
  }, [label, labelFormatter, payload, hideLabel, labelClassName, config, labelKey]);

  if (!active || !payload.length) {
    return null;
  }

  const nestLabel = payload.length === 1 && indicator !== "dot";

  return (
    <div
      ref={ref}
      role="tooltip"
      className={cn(
        "grid select-none items-start gap-1.5 rounded-lg border border-gray-6 bg-background pt-3 pb-2 text-xs shadow-md sm:w-fit md:w-fit md:max-w-[360px]",
        className,
      )}
    >
      {nestLabel ? null : tooltipLabel}
      <div className="grid gap-0.5">
        {payload.map((item: Record<string, unknown>, index: number) => {
          const key = `${nameKey || item?.name || item?.dataKey || "value"}`;
          const itemConfig = getPayloadConfigFromPayload(config, item, key);
          const itemPayload = item?.payload as Record<string, unknown> | undefined;
          const indicatorColor = color || itemPayload?.fill || item?.color;
          const dataKey =
            typeof item?.dataKey === "string" || typeof item?.dataKey === "number"
              ? item.dataKey
              : index;
          const itemName = typeof item?.name === "string" ? item.name : "";
          const itemValue =
            typeof item?.value === "number" || typeof item?.value === "string" ? item.value : null;

          return (
            <div
              key={dataKey}
              className={cn(
                "flex w-full gap-4 px-4 [&>svg]:h-2.5 [&>svg]:w-2.5 [&>svg]:text-muted-foreground",
                indicator === "dot" && "items-center",
              )}
            >
              <>
                {itemConfig?.icon ? (
                  <itemConfig.icon />
                ) : (
                  !hideIndicator && (
                    <div
                      className={cn("shrink-0 rounded-xs border-border bg-(--color-bg)", {
                        "size-2": indicator === "dot",
                        "w-1": indicator === "line",
                        "w-0 border-[1.5px] border-dashed bg-transparent": indicator === "dashed",
                        "my-0.5": nestLabel && indicator === "dashed",
                      })}
                      style={
                        {
                          "--color-bg": indicatorColor,
                          "--color-border": indicatorColor,
                        } as React.CSSProperties
                      }
                    />
                  )
                )}
                <div
                  className={cn(
                    "flex w-full justify-between gap-4 py-1 leading-none",
                    nestLabel ? "items-end" : "items-center",
                  )}
                >
                  <div className="flex items-center gap-4">
                    {nestLabel ? tooltipLabel : null}
                    {itemConfig?.subLabel && (
                      <span className="text-gray-11 text-xs capitalize">
                        {itemConfig?.subLabel}
                      </span>
                    )}

                    <span className="text-gray-12 text-xs capitalize">
                      {itemConfig?.label || itemName}
                    </span>
                  </div>
                  <div className="ml-auto">
                    {itemValue !== null && (
                      <span className="font-mono text-gray-12 tabular-nums">
                        {formatNumber(
                          typeof itemValue === "number" ? itemValue : Number(itemValue),
                        )}
                      </span>
                    )}
                  </div>
                </div>
              </>
            </div>
          );
        })}
      </div>
      {bottomExplainer}
    </div>
  );
}

// Helper to extract item config from a payload.
function getPayloadConfigFromPayload(config: ChartConfig, payload: unknown, key: string) {
  if (typeof payload !== "object" || payload === null) {
    return undefined;
  }

  const payloadPayload =
    "payload" in payload && typeof payload.payload === "object" && payload.payload !== null
      ? payload.payload
      : undefined;

  let configLabelKey: string = key;

  if (key in payload && typeof payload[key as keyof typeof payload] === "string") {
    configLabelKey = payload[key as keyof typeof payload] as string;
  } else if (
    payloadPayload &&
    key in payloadPayload &&
    typeof payloadPayload[key as keyof typeof payloadPayload] === "string"
  ) {
    configLabelKey = payloadPayload[key as keyof typeof payloadPayload] as string;
  }

  return configLabelKey in config ? config[configLabelKey] : config[key as keyof typeof config];
}

export { ChartContainer, ChartTooltip, ChartTooltipContent };
