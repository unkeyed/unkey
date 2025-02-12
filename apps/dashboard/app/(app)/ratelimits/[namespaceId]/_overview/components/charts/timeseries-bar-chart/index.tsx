"use client";

import { generateTimeseriesData } from "../../../dev-utils";
import { LogsTimeseriesBarChart } from "../bar-chart";

const timeseries = generateTimeseriesData();
export function RatelimitOverviewLogsChart() {
  return (
    <LogsTimeseriesBarChart
      data={timeseries}
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
  );
}
