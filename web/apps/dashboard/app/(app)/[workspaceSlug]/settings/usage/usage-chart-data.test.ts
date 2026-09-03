import type { DeployUsageTimeseries } from "@unkey/clickhouse";
import { describe, expect, it } from "vitest";
import type { ComputeTree } from "./compute-tree";
import { buildUsageChart, buildUsageMetricChartData } from "./usage-chart-data";

const tree: ComputeTree = {
  microCents: 0,
  projects: [
    {
      projectId: "project_1",
      name: "Acme",
      apps: [
        {
          appId: "app_1",
          name: "API",
          environments: [
            {
              environmentId: "env_1",
              name: "production",
              cpuHours: 0,
              memoryGiBHours: 0,
              egressGiB: 0,
              diskGiBHours: 0,
              microCents: 0,
            },
          ],
          cpuHours: 0,
          memoryGiBHours: 0,
          egressGiB: 0,
          diskGiBHours: 0,
          microCents: 0,
        },
      ],
      gateway: { activeKeys: 0, microCents: 0 },
      cpuHours: 0,
      memoryGiBHours: 0,
      egressGiB: 0,
      diskGiBHours: 0,
      microCents: 0,
    },
  ],
};

describe("buildUsageChart", () => {
  it("fills empty buckets and maps rows to named series", () => {
    const hour = 60 * 60 * 1000;
    const rows: DeployUsageTimeseries[] = [
      {
        time: 0,
        groupId: "project_1",
        cpuHours: 1,
        memoryGiBHours: 2,
        diskGiBHours: 3,
        egressGiB: 4,
      },
      {
        time: hour * 2,
        groupId: "project_1",
        cpuHours: 1,
        memoryGiBHours: 2,
        diskGiBHours: 3,
        egressGiB: 6,
      },
    ];

    const result = buildUsageChart({
      rows,
      metric: "egressGiB",
      groupBy: "project",
      interval: "hour",
      tree,
      start: 0,
      end: hour * 3,
    });

    expect(result.series).toHaveLength(1);
    expect(result.series[0]).toMatchObject({ key: "series0", label: "Acme" });
    expect(result.data).toEqual([
      { originalTimestamp: 0, series0: 4 },
      { originalTimestamp: hour, series0: 0 },
      { originalTimestamp: hour * 2, series0: 6 },
    ]);
    expect(result.total).toBe(10);
  });

  it("keeps a group's color when its usage rank changes", () => {
    const hour = 60 * 60 * 1000;
    const first = buildUsageChart({
      rows: [
        {
          time: 0,
          groupId: "app_1",
          cpuHours: 10,
          memoryGiBHours: 0,
          diskGiBHours: 0,
          egressGiB: 0,
        },
        {
          time: 0,
          groupId: "app_2",
          cpuHours: 5,
          memoryGiBHours: 0,
          diskGiBHours: 0,
          egressGiB: 0,
        },
      ],
      metric: "cpuHours",
      groupBy: "app",
      interval: "hour",
      tree,
      start: 0,
      end: hour,
    });
    const second = buildUsageChart({
      rows: [
        {
          time: 0,
          groupId: "app_1",
          cpuHours: 1,
          memoryGiBHours: 0,
          diskGiBHours: 0,
          egressGiB: 0,
        },
        {
          time: 0,
          groupId: "app_2",
          cpuHours: 20,
          memoryGiBHours: 0,
          diskGiBHours: 0,
          egressGiB: 0,
        },
      ],
      metric: "cpuHours",
      groupBy: "app",
      interval: "hour",
      tree,
      start: 0,
      end: hour,
    });

    expect(first.series.find((series) => series.label === "API")?.color).toBe(
      second.series.find((series) => series.label === "API")?.color,
    );
    expect(first.series[0]?.color).not.toBe(first.series[1]?.color);
  });

  it("keeps totals exact when small groups are combined", () => {
    const rows: DeployUsageTimeseries[] = Array.from({ length: 8 }, (_, index) => ({
      time: 0,
      groupId: `project_${index}`,
      cpuHours: 0,
      memoryGiBHours: 0,
      diskGiBHours: 0,
      egressGiB: index + 1,
    }));

    const result = buildUsageChart({
      rows,
      metric: "egressGiB",
      groupBy: "project",
      interval: "hour",
      tree,
      start: 0,
      end: 60 * 60 * 1000,
    });

    expect(result.hasOther).toBe(true);
    expect(result.series.at(-1)?.label).toBe("Other");
    expect(result.data[0]?.other).toBe(1);
    expect(result.total).toBe(36);
  });
});

describe("buildUsageMetricChartData", () => {
  it("aggregates groups and fills missing buckets", () => {
    const day = 24 * 60 * 60 * 1000;
    const rows: DeployUsageTimeseries[] = [
      {
        time: 0,
        groupId: "app_1",
        cpuHours: 2,
        memoryGiBHours: 0,
        diskGiBHours: 0,
        egressGiB: 0,
      },
      {
        time: 0,
        groupId: "app_2",
        cpuHours: 3,
        memoryGiBHours: 0,
        diskGiBHours: 0,
        egressGiB: 0,
      },
    ];

    expect(
      buildUsageMetricChartData({
        rows,
        metric: "cpuHours",
        interval: "day",
        start: 0,
        end: day * 2,
      }),
    ).toEqual([
      { originalTimestamp: 0, value: 5 },
      { originalTimestamp: day, value: 0 },
    ]);
  });
});
