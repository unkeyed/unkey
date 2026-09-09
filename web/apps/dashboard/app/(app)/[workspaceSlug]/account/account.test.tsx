import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import type React from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => {
  const invalidateCurrentUser = vi.fn<[], Promise<void>>();
  return {
    accessToken: "access_token" as string | undefined,
    tokenLoading: false,
    tokenError: null as Error | null,
    user: { id: "user_123" } as { id: string } | null,
    impersonator: undefined as { email: string } | undefined,
    authLoading: false,
    getAccessToken: vi.fn<[], Promise<string | undefined>>(),
    refresh: vi.fn<[], Promise<string | undefined>>(),
    getAuth: vi.fn<[], Promise<void>>(),
    refreshAuth: vi.fn<[], Promise<undefined | { error: string }>>(),
    invalidateCurrentUser,
    logManagedAuthOutcome: vi.fn(),
    utils: {
      user: {
        getCurrentUser: {
          invalidate: invalidateCurrentUser,
        },
      },
    },
    widgetTokenGetter: null as null | (() => Promise<string>),
    widgetMutationErrorHandler: null as null | ((error: unknown) => void),
  };
});

vi.mock("@workos-inc/authkit-nextjs/components", () => ({
  useAuth: () => ({
    user: mocks.user,
    impersonator: mocks.impersonator,
    loading: mocks.authLoading,
    getAuth: mocks.getAuth,
    refreshAuth: mocks.refreshAuth,
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

vi.mock("@/lib/trpc/client", () => ({
  trpc: {
    useUtils: () => mocks.utils,
  },
}));

vi.mock("@unkey/workos-widgets", () => ({
  ManagedUserWidgets: ({
    getAccessToken,
    onMutationError,
  }: {
    getAccessToken: () => Promise<string>;
    onMutationError?: (error: unknown) => void;
  }) => {
    mocks.widgetTokenGetter = getAccessToken;
    mocks.widgetMutationErrorHandler = onMutationError ?? null;
    return (
      <div data-testid="managed-widgets">
        <h2 id="profile-settings-heading">Profile</h2>
        <h2>Security</h2>
      </div>
    );
  },
}));

vi.mock("@unkey/ui", () => {
  const Wrapper = ({ children }: { children?: React.ReactNode }) => <div>{children}</div>;

  return {
    PageBody: Wrapper,
    PageContainer: ({ children }: { children?: React.ReactNode }) => (
      <div data-testid="page-container">{children}</div>
    ),
    PageHeader: Wrapper,
    PageHeaderContent: Wrapper,
    PageHeaderDescription: Wrapper,
    PageHeaderTitle: ({ children }: { children?: React.ReactNode }) => <h1>{children}</h1>,
    Skeleton: () => <div data-testid="skeleton" />,
  };
});

import { AccountUnavailable } from "./account-unavailable";
import { ManagedAccount } from "./managed-account";

afterEach(cleanup);

describe("Account", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.accessToken = "access_token";
    mocks.tokenLoading = false;
    mocks.tokenError = null;
    mocks.user = { id: "user_123" };
    mocks.impersonator = undefined;
    mocks.authLoading = false;
    mocks.getAccessToken.mockResolvedValue("access_token");
    mocks.refresh.mockResolvedValue("refreshed_access_token");
    mocks.getAuth.mockResolvedValue(undefined);
    mocks.refreshAuth.mockResolvedValue(undefined);
    mocks.invalidateCurrentUser.mockResolvedValue(undefined);
    mocks.widgetTokenGetter = null;
    mocks.widgetMutationErrorHandler = null;
  });

  it("mounts the managed account widgets with the AuthKit token getter", async () => {
    render(<ManagedAccount />);

    expect(screen.getByTestId("page-container")).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Account settings" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Profile" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Security" })).toBeTruthy();
    expect(await mocks.widgetTokenGetter?.()).toBe("access_token");
    expect(mocks.getAccessToken).toHaveBeenCalledOnce();
    expect(mocks.logManagedAuthOutcome).toHaveBeenCalledWith("widget_token", "success");
  });

  it("redacts duplicate-email errors before the account widget displays them", () => {
    render(<ManagedAccount />);
    const error = new Error("This email is not available");

    mocks.widgetMutationErrorHandler?.(error);

    expect(error.message).toBe(
      "We couldn't update your email. Try again or contact support if the problem continues.",
    );
  });

  it("shows profile and security skeletons while the AuthKit identity loads", () => {
    mocks.authLoading = true;

    render(<ManagedAccount />);

    expect(screen.getByRole("heading", { name: "Profile" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Security" })).toBeTruthy();
    expect(screen.getAllByTestId("skeleton").length).toBeGreaterThan(0);
    expect(screen.queryByTestId("managed-widgets")).toBeNull();
  });

  it("keeps profile and security skeletons visible while the widget token loads", () => {
    mocks.accessToken = undefined;
    mocks.tokenLoading = true;

    render(<ManagedAccount />);

    expect(screen.getByRole("heading", { name: "Profile" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Security" })).toBeTruthy();
    expect(screen.getAllByTestId("skeleton").length).toBeGreaterThan(0);
    expect(screen.queryByTestId("managed-widgets")).toBeNull();
  });

  it("does not refresh auth when focus returns during a managed widget flow", async () => {
    render(<ManagedAccount />);

    const restoredPage = new Event("pageshow");
    Object.defineProperty(restoredPage, "persisted", { value: true });
    await act(async () => {
      window.dispatchEvent(new Event("focus"));
      window.dispatchEvent(restoredPage);
    });

    expect(screen.getByTestId("managed-widgets")).toBeTruthy();
    expect(mocks.getAccessToken).not.toHaveBeenCalled();
    expect(mocks.refreshAuth).not.toHaveBeenCalled();
    expect(mocks.invalidateCurrentUser).not.toHaveBeenCalled();
  });

  it("ignores a normal pageshow event", async () => {
    render(<ManagedAccount />);
    const pageShow = new Event("pageshow");
    Object.defineProperty(pageShow, "persisted", { value: false });

    await act(async () => {
      window.dispatchEvent(pageShow);
    });

    expect(mocks.getAccessToken).not.toHaveBeenCalled();
    expect(mocks.refreshAuth).not.toHaveBeenCalled();
  });

  it("refreshes once on unmount as a fallback", async () => {
    const view = render(<ManagedAccount />);

    view.unmount();
    await waitFor(() => expect(mocks.refreshAuth).toHaveBeenCalledOnce());

    await act(async () => {
      window.dispatchEvent(new Event("focus"));
    });
    expect(mocks.refreshAuth).toHaveBeenCalledOnce();
  });

  it("unmounts managed controls if refreshed auth begins impersonating", () => {
    const view = render(<ManagedAccount />);
    expect(screen.getByTestId("managed-widgets")).toBeTruthy();

    mocks.impersonator = { email: "admin@example.com" };
    view.rerender(<ManagedAccount />);

    expect(screen.queryByTestId("managed-widgets")).toBeNull();
    expect(screen.getByText(/disabled while you are impersonating/i)).toBeTruthy();
  });

  it("retries token loading without rotating the session twice", async () => {
    mocks.tokenError = new Error("token unavailable");
    render(<ManagedAccount />);

    fireEvent.click(screen.getByRole("button", { name: "Retry" }));

    await waitFor(() => expect(mocks.refresh).toHaveBeenCalledOnce());
    expect(mocks.getAuth).toHaveBeenCalledOnce();
    expect(mocks.invalidateCurrentUser).toHaveBeenCalled();
    expect(mocks.refreshAuth).not.toHaveBeenCalled();
  });

  it("renders a deterministic local state without managed controls", () => {
    render(<AccountUnavailable reason="local" />);

    expect(screen.getByRole("heading", { name: "Account settings unavailable" })).toBeTruthy();
    expect(screen.getByText(/uses WorkOS authentication/i)).toBeTruthy();
    expect(screen.queryByRole("heading", { name: "Security" })).toBeNull();
  });

  it("offers a fresh sign-in when the AuthKit session is gone", () => {
    mocks.user = null;

    render(<ManagedAccount />);

    expect(screen.getByRole("heading", { name: "Your session has expired" })).toBeTruthy();
    expect(screen.getByRole("link", { name: "Sign in again" }).getAttribute("href")).toBe(
      "/auth/sign-in",
    );
    expect(screen.queryByTestId("managed-widgets")).toBeNull();
  });
});
