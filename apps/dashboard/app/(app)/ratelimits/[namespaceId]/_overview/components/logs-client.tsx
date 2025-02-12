"use client";

import { generateLatencyData, generateTimeseriesData } from "../dev-utils";
import { LogsTimeseriesBarChart } from "./charts/bar-chart";
import { LogsTimeseriesAreaChart } from "./charts/line-chart";
import { RatelimitOverviewLogsTable } from "./table/logs-table";

// Generate base timeseries data
const timeseriesData = generateTimeseriesData();

// Generate latency data with matching timestamps
const latencyData = generateLatencyData(
  timeseriesData.length, // same number of points
  {
    baseAvgLatency: 150,
    baseP99Latency: 300,
    variability: 0.3,
    trendFactor: 0.4,
  },
);

// Chart configuration for latency metrics

export const LogsClient = () => {
  return (
    <div className="flex flex-col">
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
      <RatelimitOverviewLogsTable />
    </div>
  );
};
