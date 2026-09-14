import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import React from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  accessToken: "access_token" as string | undefined,
  tokenLoading: false,
  tokenError: null as Error | null,
  user: { id: "user_123" } as { id: string } | null,
  impersonator: undefined as { email: string } | undefined,
  permissions: ["widgets:users-table:manage"] as string[] | undefined,
  authLoading: false,
  getAccessToken: vi.fn<[], Promise<string | undefined>>(),
  refresh: vi.fn<[], Promise<string | undefined>>(),
  logManagedAuthOutcome: vi.fn(),
  widgetTokenGetter: null as null | (() => Promise<string>),
}));

vi.mock("@workos-inc/authkit-nextjs/components", () => ({
  useAuth: () => ({
    user: mocks.user,
    impersonator: mocks.impersonator,
    permissions: mocks.permissions,
    loading: mocks.authLoading,
  }),
  useAccessToken: () => ({
    accessToken: mocks.accessToken,
    loading: mocks.tokenLoading,
    error: mocks.tokenError,
    getAccessToken: mocks.getAccessToken,
    refresh: mocks.refresh,
  }),
}));

vi.mock("@/lib/auth/telemetry", () => ({
  logManagedAuthOutcome: mocks.logManagedAuthOutcome,
}));

vi.mock("@/hooks/use-workspace-navigation", () => ({
  useWorkspaceNavigation: () => ({ slug: "acme" }),
}));

vi.mock("@/lib/navigation/routes", () => ({
  routes: {
    auth: {
      signIn: () => "/auth/sign-in",
    },
    settings: {
      billing: () => "/acme/settings/billing",
    },
  },
}));

vi.mock("@unkey/workos-widgets", () => ({
  ManagedUsersWidget: ({ getAccessToken }: { getAccessToken: () => Promise<string> }) => {
    mocks.widgetTokenGetter = getAccessToken;
    return <div data-testid="managed-users-widget">Managed users</div>;
  },
}));

vi.mock("@unkey/ui", () => {
  const Empty = ({ children }: { children?: React.ReactNode }) => <div>{children}</div>;
  Empty.Title = ({ children }: { children?: React.ReactNode }) => <h2>{children}</h2>;
  Empty.Description = ({ children }: { children?: React.ReactNode }) => <p>{children}</p>;
  Empty.Actions = ({ children }: { children?: React.ReactNode }) => <div>{children}</div>;

  return {
    Button: ({
      children,
      loading: _loading,
      render,
      ...props
    }: React.ButtonHTMLAttributes<HTMLButtonElement> & {
      loading?: boolean;
      render?: React.ReactElement;
    }) =>
      render ? React.cloneElement(render, props, children) : <button {...props}>{children}</button>,
    Empty,
    Skeleton: () => <div data-testid="skeleton" />,
  };
});

import { ManagedTeam } from "./managed-team";

afterEach(cleanup);

describe("ManagedTeam", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.accessToken = "access_token";
    mocks.tokenLoading = false;
    mocks.tokenError = null;
    mocks.user = { id: "user_123" };
    mocks.impersonator = undefined;
    mocks.permissions = ["widgets:users-table:manage"];
    mocks.authLoading = false;
    mocks.getAccessToken.mockResolvedValue("access_token");
    mocks.refresh.mockResolvedValue("refreshed_access_token");
    mocks.widgetTokenGetter = null;
  });

  it("renders WorkOS user management with the AuthKit token getter", async () => {
    render(<ManagedTeam team />);

    expect(screen.getByRole("heading", { name: "Members" })).toBeTruthy();
    expect(screen.getByTestId("managed-users-widget")).toBeTruthy();
    expect(await mocks.widgetTokenGetter?.()).toBe("access_token");
    expect(mocks.logManagedAuthOutcome).toHaveBeenCalledWith("widget_token", "success");
  });

  it("shows the managed-team skeleton while AuthKit or its token is loading", () => {
    mocks.authLoading = true;
    const view = render(<ManagedTeam team />);

    expect(screen.getAllByTestId("skeleton").length).toBeGreaterThan(0);
    expect(screen.queryByTestId("managed-users-widget")).toBeNull();

    mocks.authLoading = false;
    mocks.accessToken = undefined;
    mocks.tokenLoading = true;
    view.rerender(<ManagedTeam team />);

    expect(screen.getAllByTestId("skeleton").length).toBeGreaterThan(0);
    expect(screen.queryByTestId("managed-users-widget")).toBeNull();
  });

  it("preserves the team-plan entitlement gate", () => {
    render(<ManagedTeam team={false} />);

    expect(
      screen.getByRole("heading", { name: "Upgrade Your Plan to Add Team Members" }),
    ).toBeTruthy();
    expect(screen.getByRole("link", { name: "Upgrade" }).getAttribute("href")).toBe(
      "/acme/settings/billing",
    );
    expect(screen.queryByTestId("managed-users-widget")).toBeNull();
  });

  it("does not mount administrative controls without a user or during impersonation", () => {
    mocks.user = null;
    const view = render(<ManagedTeam team />);

    expect(screen.getByRole("heading", { name: "Your session has expired" })).toBeTruthy();
    expect(screen.queryByTestId("managed-users-widget")).toBeNull();

    mocks.user = { id: "user_123" };
    mocks.impersonator = { email: "admin@example.com" };
    view.rerender(<ManagedTeam team />);

    expect(screen.getByRole("heading", { name: "Team management unavailable" })).toBeTruthy();
    expect(screen.queryByTestId("managed-users-widget")).toBeNull();
  });

  it("requires the WorkOS user-management permission", () => {
    mocks.permissions = [];

    render(<ManagedTeam team />);

    expect(screen.getByRole("heading", { name: "Admin access required" })).toBeTruthy();
    expect(screen.queryByTestId("managed-users-widget")).toBeNull();
  });

  it("retries a failed widget token without a page reload", async () => {
    mocks.tokenError = new Error("token unavailable");
    render(<ManagedTeam team />);

    fireEvent.click(screen.getByRole("button", { name: "Retry" }));

    await waitFor(() => expect(mocks.refresh).toHaveBeenCalledOnce());
    expect(mocks.logManagedAuthOutcome).toHaveBeenCalledWith("widget_token", "success");
  });
});
