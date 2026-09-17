import type { ProjectApp } from "@/lib/collections/deploy/projects";
import { routes } from "@/lib/navigation/routes";
import { act, cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { type ReactNode, useState } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@unkey/ui", () => ({
  InfoTooltip: ({ children, content }: { children: ReactNode; content: ReactNode }) => {
    const [open, setOpen] = useState(false);
    return (
      <div onMouseEnter={() => setOpen(true)}>
        {children}
        {open ? <div data-testid="tooltip">{content}</div> : null}
      </div>
    );
  },
}));

const { AppDetailTooltip, AppRow } = await import("./app-row");

const app = (deployedAt: number): ProjectApp => ({
  id: "app_1",
  name: "riceipt",
  customDomain: null,
  headlineDeployment: {
    id: "dpl_1",
    status: "ready",
    commitMessage: null,
    branch: null,
    deployedAt,
  },
});

describe("AppDetailTooltip", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-01T12:00:00.000Z"));
  });

  afterEach(() => {
    cleanup();
    vi.useRealTimers();
  });

  it("gives the tooltip it opens the same age as the row under it", () => {
    const project = app(Date.now() - 59_500);
    const appHref = routes.projects.apps.overview({
      workspaceSlug: "acme",
      projectId: "proj_1",
      appId: project.id,
    });
    render(
      <AppDetailTooltip app={project}>
        <AppRow app={project} href={appHref} />
      </AppDetailTooltip>,
    );

    act(() => {
      vi.advanceTimersByTime(900);
    });
    const row = screen.getByRole("link");
    const rowAge = within(row).getByText(/ago$/).textContent;

    fireEvent.mouseEnter(row);

    expect(within(screen.getByTestId("tooltip")).getByText(`deployed ${rowAge}`)).toBeTruthy();
  });
});
