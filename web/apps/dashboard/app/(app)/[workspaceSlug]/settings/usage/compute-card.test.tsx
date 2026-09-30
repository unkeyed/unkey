import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import React from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ComputeCard } from "./compute-card";
import { buildComputeTree } from "./compute-tree";

vi.stubGlobal("React", React);
vi.mock("@/lib/trpc/client", () => ({
  trpc: { billing: { queryDeployUsageTimeseries: { useQuery: () => ({ data: [] }) } } },
}));
vi.mock("./spend-bar-chart", () => ({ SPEND_BAR_CHART_HEIGHT: 200, SpendBarChart: () => null }));

afterEach(cleanup);

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
    render(<ComputeCard tree={tree} />);

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
