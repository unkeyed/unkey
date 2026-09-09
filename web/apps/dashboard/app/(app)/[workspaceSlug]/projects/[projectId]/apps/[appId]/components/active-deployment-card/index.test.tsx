import { render, screen } from "@testing-library/react";
import type { LastPodFailure } from "@unkey/db/src/schema";
import type { PropsWithChildren, ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { LastPodFailureBadge, shouldShowLastExit } from ".";

vi.mock("@unkey/ui", () => ({
  Badge: ({ children }: PropsWithChildren) => <span>{children}</span>,
  InfoTooltip: ({ children, content }: PropsWithChildren<{ content: ReactNode }>) => (
    <div>
      {children}
      {content}
    </div>
  ),
  TimestampInfo: () => null,
}));

describe("deployment failure badges", () => {
  it("shows historical failure context after the deployment is ready", () => {
    const failure: LastPodFailure = {
      podUid: "pod-uid",
      podName: "api-7ddf9",
      regionId: "us-east-1",
      reason: "Evicted",
      message: "The node was low on ephemeral-storage.",
      observedAt: Date.UTC(2026, 8, 8, 12, 30),
    };

    render(<LastPodFailureBadge failure={failure} />);

    expect(screen.getByText("Last instance failure · Evicted")).toBeTruthy();
    expect(screen.getByText("Historical instance failure")).toBeTruthy();
    expect(
      screen.getByText("This is the last observed failure, not the deployment's current health."),
    ).toBeTruthy();
    expect(screen.getByText("api-7ddf9")).toBeTruthy();
    expect(screen.getByText("The node was low on ephemeral-storage.")).toBeTruthy();
  });

  it("clears an active startup failure after recovery", () => {
    const lastExit = {
      restartCount: 1,
      exitCode: 1,
      signal: null,
      reason: "Error",
      finishedAt: Date.UTC(2026, 8, 8, 12, 0),
      statusReason: null,
      statusMessage: null,
    };

    expect(shouldShowLastExit({ lastExit, status: "deploying" })).toBe(true);
    expect(shouldShowLastExit({ lastExit, status: "ready" })).toBe(false);
  });
});
