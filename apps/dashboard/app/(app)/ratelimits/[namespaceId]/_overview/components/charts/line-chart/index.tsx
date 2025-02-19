import { calculateTimePoints } from "@/components/logs/chart/utils/calculate-timepoints";
import { formatTimestampLabel } from "@/components/logs/chart/utils/format-timestamp";
import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";
import { useEffect, useState } from "react";
import {
  Area,
  AreaChart,
  CartesianGrid,
  ReferenceArea,
  ResponsiveContainer,
  YAxis,
} from "recharts";
import { LogsTimeseriesAreaChartError } from "./components/logs-chart-error";
import { LogsTimeseriesAreaChartLoading } from "./components/logs-chart-loading";

const latencyFormatter = new Intl.NumberFormat("en-US", {
  maximumFractionDigits: 2,
  minimumFractionDigits: 2,
});

type Selection = {
  start: string | number;
  end: string | number;
  startTimestamp?: number;
  endTimestamp?: number;
};

type TimeseriesData = {
  originalTimestamp: number;
  avgLatency: number;
  p99Latency: number;
  [key: string]: any;
};

interface LogsTimeseriesAreaChartProps {
  data?: TimeseriesData[];
  config: ChartConfig;
  onSelectionChange?: (selection: { start: number; end: number }) => void;
  isLoading?: boolean;
  isError?: boolean;
  enableSelection?: boolean;
}

const formatTimestampTooltip = (timestamp: number): string => {
  return new Date(timestamp).toLocaleString();
};

export const LogsTimeseriesAreaChart: React.FC<LogsTimeseriesAreaChartProps> = ({
  data = [],
  config,
  onSelectionChange,
  isLoading,
  isError,
  enableSelection = false,
}) => {
  const [selection, setSelection] = useState<Selection>({ start: "", end: "" });
  const [isDarkMode, setIsDarkMode] = useState(false);

  useEffect(() => {
    const darkModeMediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
    setIsDarkMode(darkModeMediaQuery.matches);

    const handleThemeChange = (e: MediaQueryListEvent) => setIsDarkMode(e.matches);
    darkModeMediaQuery.addEventListener("change", handleThemeChange);

    return () => darkModeMediaQuery.removeEventListener("change", handleThemeChange);
  }, []);

  const getThemeColor = (lightColor: string, darkColor: string) => {
    return isDarkMode ? darkColor : lightColor;
  };

  const chartConfig = {
    avgLatency: {
      color: getThemeColor("hsl(var(--accent-11))", "hsl(var(--accent-11))"),
      label: config.avgLatency.label,
    },
    p99Latency: {
      color: getThemeColor("hsl(var(--warning-10))", "hsl(var(--warning-11))"),
      label: config.p99Latency.label,
    },
  };

  const handleMouseDown = (e: any) => {
    if (!enableSelection) {
      return;
    }
    const timestamp = e?.activePayload?.[0]?.payload?.originalTimestamp;
    setSelection({
      start: e.activeLabel,
      end: e.activeLabel,
      startTimestamp: timestamp,
      endTimestamp: timestamp,
    });
  };

  const handleMouseMove = (e: any) => {
    if (!enableSelection || !selection.start) {
      return;
    }
    const timestamp = e?.activePayload?.[0]?.payload?.originalTimestamp;
    setSelection((prev) => ({
      ...prev,
      end: e.activeLabel,
      endTimestamp: timestamp,
    }));
  };

  const handleMouseUp = () => {
    if (!enableSelection) {
      return;
    }
    if (selection.start && selection.end && onSelectionChange) {
      if (!selection.startTimestamp || !selection.endTimestamp) {
        return;
      }
      const [start, end] = [selection.startTimestamp, selection.endTimestamp].sort((a, b) => a - b);
      onSelectionChange({ start, end });
    }
    setSelection({
      start: "",
      end: "",
      startTimestamp: undefined,
      endTimestamp: undefined,
    });
  };

  if (isError) {
    return <LogsTimeseriesAreaChartError />;
  }
  if (isLoading) {
    return <LogsTimeseriesAreaChartLoading />;
  }

  // Calculate metrics
  const minLatency = data.length > 0 ? Math.min(...data.map((d) => d.avgLatency)) : 0;
  const maxLatency = data.length > 0 ? Math.max(...data.map((d) => d.p99Latency)) : 0;
  const avgLatency =
    data.length > 0 ? data.reduce((acc, curr) => acc + curr.avgLatency, 0) / data.length : 0;

  return (
    <div className="flex flex-col h-full">
      <div className="pl-5 pt-4 py-3 pr-10 w-full flex justify-between font-sans items-start gap-10">
        <div className="flex flex-col gap-1">
          <div className="text-accent-10 text-[11px] leading-4">DURATION</div>
          <div className="text-accent-12 text-[18px] font-semibold leading-7">
            {minLatency} - {maxLatency}ms
          </div>
        </div>

        <div className="flex gap-10 items-center">
          <div className="flex flex-col gap-1">
            <div className="flex gap-2 items-center">
              <div className="bg-accent-8 rounded h-[10px] w-1" />
              <div className="text-accent-10 text-[11px] leading-4">AVG</div>
            </div>
            <div className="text-accent-12 text-[18px] font-semibold leading-7">
              {latencyFormatter.format(avgLatency)}ms
            </div>
          </div>
          <div className="flex flex-col gap-1">
            <div className="flex gap-2 items-center">
              <div className="bg-orange-9 rounded h-[10px] w-1" />
              <div className="text-accent-10 text-[11px] leading-4">P99</div>
            </div>
            <div className="text-accent-12 text-[18px] font-semibold leading-7">
              {latencyFormatter.format(maxLatency)}ms
            </div>
          </div>
        </div>
      </div>

      <div className="flex-1 min-h-0">
        <ResponsiveContainer width="100%" height="100%">
          <ChartContainer config={config}>
            <AreaChart
              data={data}
              margin={{ top: 0, right: 0, bottom: 0, left: 0 }}
              onMouseDown={handleMouseDown}
              onMouseMove={handleMouseMove}
              onMouseUp={handleMouseUp}
              onMouseLeave={handleMouseUp}
            >
              <defs>
                <linearGradient id="avgGradient" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor={chartConfig.avgLatency.color} stopOpacity={0.2} />
                  <stop offset="95%" stopColor={chartConfig.avgLatency.color} stopOpacity={0} />
                </linearGradient>
                <linearGradient id="p99Gradient" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor={chartConfig.p99Latency.color} stopOpacity={0.3} />
                  <stop offset="95%" stopColor={chartConfig.p99Latency.color} stopOpacity={0} />
                </linearGradient>
              </defs>

              <YAxis domain={["auto", (dataMax: number) => dataMax * 1.1]} hide />
              <CartesianGrid
                horizontal
                vertical={false}
                strokeDasharray="3 3"
                stroke="hsl(var(--gray-6))"
                strokeOpacity={0.3}
                strokeWidth={1}
              />
              <ChartTooltip
                position={{ y: 50 }}
                isAnimationActive
                wrapperStyle={{ zIndex: 1000 }}
                cursor={{
                  stroke: "hsl(var(--accent-3))",
                  strokeWidth: 1,
                  strokeDasharray: "5 5",
                  strokeOpacity: 0.7,
                }}
                content={({ active, payload, label }) => {
                  if (!active || !payload?.length) {
                    return null;
                  }
                  return (
                    <ChartTooltipContent
                      payload={payload}
                      label={label}
                      active={active}
                      className="rounded-lg shadow-lg border border-gray-4"
                      labelFormatter={(_, tooltipPayload) => {
                        const originalTimestamp = tooltipPayload[0]?.payload?.originalTimestamp;
                        return originalTimestamp ? (
                          <div>
                            <span className="font-mono text-accent-9 text-xs px-4">
                              {formatTimestampTooltip(originalTimestamp)}
                            </span>
                          </div>
                        ) : (
                          ""
                        );
                      }}
                    />
                  );
                }}
              />
              <Area
                type="monotone"
                dataKey="avgLatency"
                stroke={chartConfig.avgLatency.color}
                fill="url(#avgGradient)"
                fillOpacity={1}
                strokeWidth={2}
              />
              <Area
                type="monotone"
                dataKey="p99Latency"
                stroke={chartConfig.p99Latency.color}
                strokeWidth={2}
                fill="url(#p99Gradient)"
                fillOpacity={1}
              />
              {enableSelection && selection.start && selection.end && (
                <ReferenceArea
                  x1={Math.min(Number(selection.start), Number(selection.end))}
                  x2={Math.max(Number(selection.start), Number(selection.end))}
                  fill="hsl(var(--chart-selection))"
                  fillOpacity={0.3}
                />
              )}
            </AreaChart>
          </ChartContainer>
        </ResponsiveContainer>
      </div>

      <div className="h-8 border-t border-b border-gray-4 px-1 py-2 text-accent-9 font-mono text-xxs w-full flex justify-between border-t-gray-2">
        {data
          ? calculateTimePoints(
              data[0]?.originalTimestamp ?? Date.now(),
              data.at(-1)?.originalTimestamp ?? Date.now(),
            ).map((time, i) => (
              // biome-ignore lint/suspicious/noArrayIndexKey: <explanation>
              <div key={i} className="z-10">
                {formatTimestampLabel(time)}
              </div>
            ))
          : null}
      </div>
    </div>
  );
};
