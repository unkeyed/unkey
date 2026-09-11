import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import type { ComponentProps } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Crumb } from "./crumb";
import { WorkspaceCrumb } from "./workspace-crumb";

const mocks = vi.hoisted(() => ({
  query: {
    data: [{ orgId: "org_disabled", name: "Disabled workspace" }],
    isError: false,
    isLoading: false,
    isFetching: false,
    refetch: vi.fn(),
  },
  mutate: vi.fn(),
  sessionCookie: vi.fn(),
  lastOrgCookie: vi.fn(),
  error: vi.fn(),
}));

vi.mock("@/hooks/use-workspace-navigation", () => ({
  useWorkspaceNavigation: () => ({ orgId: "org_current", name: "Current" }),
}));
vi.mock("@/lib/auth/cookies-actions", () => ({
  setSessionCookie: mocks.sessionCookie,
  setLastUsedOrgCookie: mocks.lastOrgCookie,
}));
vi.mock("@/lib/trpc/client", () => ({
  trpc: {
    workspace: { listAvailable: { useQuery: () => mocks.query } },
    user: {
      switchOrg: {
        useMutation: (options: { onError: () => void }) => ({
          isLoading: false,
          mutate: (orgId: string) => {
            mocks.mutate(orgId);
            options.onError();
          },
        }),
      },
    },
  },
}));
vi.mock("@unkey/ui", () => ({
  toast: { error: mocks.error },
  Button: ({ children, onClick, disabled }: ComponentProps<"button">) => (
    <button type="button" onClick={onClick} disabled={disabled}>
      {children}
    </button>
  ),
}));
vi.mock("./crumb", () => ({
  Crumb: ({ items, listStatus }: ComponentProps<typeof Crumb>) => (
    <div>
      {listStatus}
      {items.map((item) => (
        <button type="button" key={item.id} onClick={item.onClick}>
          {item.label}
        </button>
      ))}
    </div>
  ),
}));

afterEach(cleanup);
beforeEach(() => {
  vi.clearAllMocks();
  mocks.query.isError = false;
  mocks.query.isLoading = false;
  mocks.query.data = [{ orgId: "org_disabled", name: "Disabled workspace" }];
});

describe("workspace selector", () => {
  it("keeps disabled workspaces selectable but never writes cookies when the server rejects the switch", () => {
    render(<WorkspaceCrumb href="/" />);
    fireEvent.click(screen.getByRole("button", { name: "Disabled workspace" }));
    expect(mocks.mutate).toHaveBeenCalledWith("org_disabled");
    expect(mocks.error).toHaveBeenCalled();
    expect(mocks.sessionCookie).not.toHaveBeenCalled();
    expect(mocks.lastOrgCookie).not.toHaveBeenCalled();
  });

  it("shows a retry action rather than stale workspaces or an empty state on failure", () => {
    mocks.query.isError = true;
    render(<WorkspaceCrumb href="/" />);
    expect(screen.getByRole("alert").textContent).toContain("Unable to load workspaces");
    expect(screen.queryByText("Disabled workspace")).toBeNull();
    expect(screen.queryByText("No workspaces found")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(mocks.query.refetch).toHaveBeenCalledOnce();
  });

  it("distinguishes an empty result from loading", () => {
    mocks.query.data = [];
    const view = render(<WorkspaceCrumb href="/" />);
    expect(screen.getByRole("status").textContent).toBe("No workspaces found");
    mocks.query.isLoading = true;
    view.rerender(<WorkspaceCrumb href="/" />);
    expect(screen.getByRole("status").textContent).toBe("Loading workspaces...");
  });
});
