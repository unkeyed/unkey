import { act, cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { type AppDeployment, DeploymentMeta, useDeploymentPhrase } from "./deployment-meta";

const deployment = (overrides: Partial<AppDeployment> = {}): AppDeployment => ({
  id: "dpl_1",
  status: "ready",
  commitMessage: null,
  branch: null,
  deployedAt: Date.now(),
  ...overrides,
});

function Phrase({ deployment }: { deployment: AppDeployment }) {
  return <span>{useDeploymentPhrase(deployment)}</span>;
}

describe("DeploymentMeta", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-01T12:00:00.000Z"));
  });

  afterEach(() => {
    cleanup();
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

  it("never puts a deployment in the future when the browser clock runs behind", () => {
    render(
      <DeploymentMeta
        deployment={deployment({ status: "building", deployedAt: Date.now() + 25_000 })}
      />,
    );

    expect(screen.getByText("now")).toBeTruthy();
  });

  it("gives a tooltip opened later the same age as the row it covers", () => {
    const app = deployment({ deployedAt: Date.now() - 59_500 });
    render(<DeploymentMeta deployment={app} />);
    const rowAge = screen.getByText(/ago$/).textContent;

    act(() => {
      vi.advanceTimersByTime(900);
    });
    render(<Phrase deployment={app} />);

    expect(screen.getByText(`deployed ${rowAge}`)).toBeTruthy();
  });
});
