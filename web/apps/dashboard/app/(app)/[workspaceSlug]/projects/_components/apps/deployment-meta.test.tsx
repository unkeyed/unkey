import { act, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { type AppDeployment, DeploymentMeta } from "./deployment-meta";

const deployment = (overrides: Partial<AppDeployment> = {}): AppDeployment => ({
  id: "dpl_1",
  status: "ready",
  commitMessage: null,
  branch: null,
  deployedAt: Date.now(),
  ...overrides,
});

describe("DeploymentMeta", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-01T12:00:00.000Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("ages the label without anything else re-rendering it", () => {
    render(<DeploymentMeta deployment={deployment({ deployedAt: Date.now() - 61_000 })} />);
    expect(screen.getByText("1 min ago")).toBeTruthy();

    act(() => {
      vi.advanceTimersByTime(60_000);
    });

    expect(screen.getByText("2 min ago")).toBeTruthy();
  });

  it("never puts a deployment that landed between ticks in the future", () => {
    render(<DeploymentMeta deployment={deployment({ deployedAt: Date.now() + 9_000 })} />);
    expect(screen.getByText("now")).toBeTruthy();
  });
});
