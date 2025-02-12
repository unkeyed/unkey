import { generateLatencyData, generateTimeseriesData } from "../../dev-utils";
import { LogsTimeseriesBarChart } from "./bar-chart";
import { LogsTimeseriesAreaChart } from "./line-chart";
// Generate base timeseries data
const timeseriesData = generateTimeseriesData();

// Generate latency data with matching timestamps
const latencyData = generateLatencyData(timeseriesData.length, {
  baseAvgLatency: 150,
  baseP99Latency: 300,
  variability: 0.3,
  trendFactor: 0.4,
});
export const RatelimitOverviewLogsCharts = () => {
  return (
    <div className="flex w-full h-[320px]">
      <div className="w-1/2 border-r border-gray-4">
        <LogsTimeseriesBarChart
          data={timeseriesData}
          config={{
            success: {
              label: "Passed",
              color: "hsl(var(--accent-4))",
            },
            error: {
              label: "Blocked",
              color: "hsl(var(--orange-9))",
            },
          }}
        />
      </div>
      <div className="w-1/2">
        <LogsTimeseriesAreaChart
          data={latencyData}
          config={{
            avgLatency: {
              color: "hsl(var(--accent-8))",
              label: "Average Latency",
            },
            p99Latency: {
              color: "hsl(var(--warning-11))",
              label: "P99 Latency",
            },
          }}
        />
      </div>
    </div>
  );
};
