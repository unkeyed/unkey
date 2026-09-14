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

afterEach(() => {
  cleanup();
  window.getSelection()?.removeAllRanges();
});

describe("deleted billing IDs", () => {
  it("shows selectable IDs only for deleted resources and preserves keyboard expansion", () => {
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
          projectName: "Deleted project",
          appId: "app_live",
          activeKeys: 0,
          grossMicroCents: 0,
        },
        { projectId: "", projectName: null, appId: "", activeKeys: 0, grossMicroCents: 0 },
      ],
    });
    render(<ComputeCard tree={tree} />);
    for (const id of ["proj_deleted", "app_deleted", "env_deleted"]) {
      const label = screen.getByText(id);
      expect(label.title).toBe(id);
      expect(label.classList.contains("select-text")).toBe(true);
    }
    expect(screen.queryByText("proj_live")).toBeNull();
    expect(screen.getByText("Unattributed")).toBeTruthy();
    const button = screen.getByRole("button", { name: /proj_deleted/ });
    fireEvent.click(button);
    expect(button.getAttribute("aria-expanded")).toBe("true");

    const range = document.createRange();
    range.selectNodeContents(screen.getByText("proj_deleted"));
    window.getSelection()?.addRange(range);
    fireEvent.click(button, { detail: 1 });
    expect(button.getAttribute("aria-expanded")).toBe("true");
    fireEvent.click(button, { detail: 0 });
    expect(button.getAttribute("aria-expanded")).toBe("false");
  });
});
