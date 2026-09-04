import { describe, expect, it } from "vitest";
import type { ComputeTree, UsageProject } from "./compute-tree";
import { buildSpendSeries } from "./spend-series";

const DAY = 24 * 60 * 60 * 1000;
const START = Date.UTC(2026, 8, 1);

function project(projectId: string, name: string): UsageProject {
  return {
    projectId,
    name,
    apps: [],
    gateway: { activeKeys: 0, microCents: 0 },
    cpuHours: 0,
    memoryGiBHours: 0,
    egressGiB: 0,
    diskGiBHours: 0,
    microCents: 0,
  };
}

const tree: ComputeTree = {
  projects: [project("proj_a", "Platform"), project("", "Unattributed")],
  microCents: 0,
};

function row(groupId: string, time: number, egressGiB: number) {
  return { groupId, time, cpuHours: 0, memoryGiBHours: 0, egressGiB, diskGiBHours: 0 };
}

describe("buildSpendSeries", () => {
  it("prices each project per day and zero-fills the rest of the range", () => {
    const { points, series } = buildSpendSeries({
      tree,
      rows: [row("proj_a", START, 2), row("proj_a", START, 1), row("", START + DAY, 4)],
      start: START,
      end: START + 3 * DAY,
    });

    expect(series).toEqual([
      { key: "proj_a", label: "Platform", color: "hsl(var(--info-9))" },
      { key: "unattributed", label: "Unattributed", color: "hsl(var(--feature-9))" },
    ]);
    expect(points).toEqual([
      { time: START, proj_a: 15, unattributed: 0 },
      { time: START + DAY, proj_a: 0, unattributed: 20 },
      { time: START + 2 * DAY, proj_a: 0, unattributed: 0 },
    ]);
  });

  it("drops rows outside the range or for unknown projects", () => {
    const { points } = buildSpendSeries({
      tree,
      rows: [row("proj_a", START - DAY, 1), row("proj_other", START, 1)],
      start: START,
      end: START + DAY,
    });

    expect(points).toEqual([{ time: START, proj_a: 0, unattributed: 0 }]);
  });
});
