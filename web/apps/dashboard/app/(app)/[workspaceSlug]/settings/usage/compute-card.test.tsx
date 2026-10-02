import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import React from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ComputeCard } from "./compute-card";
import { buildComputeTree } from "./compute-tree";

vi.stubGlobal("React", React);
vi.mock("@/lib/trpc/client", () => ({
  trpc: { billing: { queryDeployUsageTimeseries: { useQuery: () => ({ data: [] }) } } },
}));
vi.mock("./spend-bar-chart", () => ({
  SPEND_BAR_CHART_HEIGHT: 200,
  SpendBarChart: ({ data }: { data: unknown[] }) => (
    <div data-testid="spend-chart" data-point-count={data.length} />
  ),
}));

afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

describe("monthly spend chart", () => {
  it("includes every day in the current month", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date(Date.UTC(2026, 9, 1, 15, 42)));
    const tree = buildComputeTree({
      usage: [
        {
          projectId: "proj_1",
          projectName: "Platform",
          appId: "app_1",
          appName: "API",
          environmentId: "env_1",
          environmentSlug: "production",
          cpuSeconds: 1,
          memoryGiBHours: 0,
          egressGiB: 0,
          diskGiBHours: 0,
          grossMicroCents: 1,
        },
      ],
      gateway: [],
    });

    render(<ComputeCard tree={tree} period="current" />);

    expect(screen.getByTestId("spend-chart").getAttribute("data-point-count")).toBe("31");
  });
});

describe("deleted billing IDs", () => {
  it("keeps IDs out of the visible label but readable by assistive tech", () => {
    const tree = buildComputeTree({
      usage: [
        {
          projectId: "proj_deleted",
          projectName: null,
          appId: "app_deleted",
          appName: null,
          environmentId: "env_deleted",
          environmentSlug: null,
          cpuSeconds: 0,
          memoryGiBHours: 0,
          egressGiB: 0,
          diskGiBHours: 0,
          grossMicroCents: 0,
        },
      ],
      gateway: [
        {
          projectId: "proj_live",
          projectName: "Checkout",
          appId: "app_live",
          activeKeys: 0,
          grossMicroCents: 0,
        },
        { projectId: "", projectName: null, appId: "", activeKeys: 0, grossMicroCents: 0 },
      ],
    });
    render(<ComputeCard tree={tree} period="current" />);

    for (const id of ["proj_deleted", "app_deleted", "env_deleted"]) {
      expect(screen.getByText(`, ${id}`).classList.contains("sr-only")).toBe(true);
      expect(screen.queryByText(id)).toBeNull();
    }
    expect(screen.getByText("Deleted project")).toBeTruthy();
    expect(screen.getByText("Deleted app")).toBeTruthy();
    expect(screen.getByText("Deleted environment")).toBeTruthy();
    expect(screen.getByText("Checkout")).toBeTruthy();
    expect(screen.queryByText("proj_live")).toBeNull();
    expect(screen.getByText("Unattributed")).toBeTruthy();

    const button = screen.getByRole("button", { name: /proj_deleted/ });
    fireEvent.click(button);
    expect(button.getAttribute("aria-expanded")).toBe("true");
    fireEvent.click(button);
    expect(button.getAttribute("aria-expanded")).toBe("false");
  });
});
