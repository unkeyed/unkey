import type { PortalState } from "@/lib/portal/use-portal";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import type { Portal } from "@unkey/api/models/components";
import type { ButtonHTMLAttributes, ReactNode } from "react";
import { type Mock, beforeEach, describe, expect, it, vi } from "vitest";
import { PortalLifecyclePage } from "./portal-lifecycle-page";

const mocks = vi.hoisted(
  (): {
    portalState: PortalState;
    updateMutation: { mutate: Mock; isLoading: boolean };
    invalidateQueries: Mock;
  } => ({
    portalState: { status: "loading" },
    updateMutation: { mutate: vi.fn(), isLoading: false },
    invalidateQueries: vi.fn(),
  }),
);

vi.mock("@/lib/portal/use-portal", () => ({
  usePortal: () => mocks.portalState,
  useUpdatePortal: () => mocks.updateMutation,
  portalQueryKey: (keyAuthId: string) => ["portal", keyAuthId],
}));

vi.mock("@tanstack/react-query", () => ({
  useQueryClient: () => ({ invalidateQueries: mocks.invalidateQueries }),
}));

vi.mock("./portal-config", () => ({
  PortalConfig: ({
    portal: configPortal,
    keyAuthId,
  }: {
    portal: Portal;
    keyAuthId: string;
  }) => <div data-testid="portal-config">{`${configPortal.slug}|${keyAuthId}`}</div>,
}));

vi.mock("./integrate-dialog", () => ({
  IntegrateDialog: () => null,
}));

vi.mock("./setup-hero", () => ({
  SetupHero: () => <div data-testid="setup-hero">Enable Customer portal</div>,
}));

vi.mock("@unkey/icons", () => ({
  BookBookmark: () => null,
  CircleWarning: () => null,
  TriangleWarning2: () => null,
}));

vi.mock("@unkey/ui", () => {
  const passthrough = ({ children }: { children?: ReactNode }) => <div>{children}</div>;
  return {
    Button: ({
      children,
      loading: _loading,
      loadingLabel: _loadingLabel,
      ...props
    }: ButtonHTMLAttributes<HTMLButtonElement> & {
      loading?: boolean;
      loadingLabel?: string;
    }) => <button {...props}>{children}</button>,
    AlertBanner: passthrough,
    AlertBannerActions: passthrough,
    AlertBannerDescription: passthrough,
    AlertBannerTitle: passthrough,
    Skeleton: () => <div />,
    PageBody: passthrough,
    PageContainer: passthrough,
    PageHeader: passthrough,
    PageHeaderActions: passthrough,
    PageHeaderContent: passthrough,
    PageHeaderTitle: passthrough,
  };
});

const portal: Portal = {
  id: "portal_123",
  slug: "acme",
  displayName: "Acme",
  enabled: false,
  mapping: { type: "keyspace", id: "ks_123" },
  createdAt: 0,
};

const onRetryKeyAuthId = vi.fn();

function renderPage(overrides?: {
  keyAuthId?: string | undefined;
  keyAuthIdLoading?: boolean;
  keyAuthIdError?: boolean;
}) {
  return render(
    <PortalLifecyclePage
      resourceName="Acme"
      keyAuthId={"keyAuthId" in (overrides ?? {}) ? overrides?.keyAuthId : "ks_123"}
      keyAuthIdLoading={overrides?.keyAuthIdLoading ?? false}
      keyAuthIdError={overrides?.keyAuthIdError ?? false}
      onRetryKeyAuthId={onRetryKeyAuthId}
    />,
  );
}

describe("PortalLifecyclePage", () => {
  // Vitest globals are off here, so testing-library does not auto-unmount.
  beforeEach(() => {
    cleanup();
    mocks.portalState = { status: "loading" };
    mocks.updateMutation.isLoading = false;
    mocks.updateMutation.mutate.mockClear();
    mocks.invalidateQueries.mockClear();
    onRetryKeyAuthId.mockClear();
  });

  it("renders a retry panel rather than the setup hero when the read fails", () => {
    mocks.portalState = { status: "error", message: "upstream exploded" };

    renderPage();

    expect(screen.getByText("upstream exploded")).toBeDefined();
    expect(screen.queryByTestId("setup-hero")).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(mocks.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["portal", "ks_123"] });
  });

  it("renders the error state when the keyspace finished resolving as undefined", () => {
    mocks.portalState = { status: "notConfigured" };

    renderPage({ keyAuthId: undefined, keyAuthIdLoading: false });

    expect(screen.getByText(/no keyspace/i)).toBeDefined();
    expect(screen.queryByTestId("setup-hero")).toBeNull();
    // Retrying cannot help without a keyspace, so no retry action is offered.
    expect(screen.queryByRole("button", { name: "Retry" })).toBeNull();
  });

  it("offers a retry rather than the dead-end message when the keyspace lookup fails", () => {
    mocks.portalState = { status: "notConfigured" };

    renderPage({ keyAuthId: undefined, keyAuthIdLoading: false, keyAuthIdError: true });

    // A failed lookup must never read as "this API has no keyspace": the API
    // may well have one we simply could not read.
    expect(screen.queryByText(/no keyspace/i)).toBeNull();
    expect(screen.queryByTestId("setup-hero")).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(onRetryKeyAuthId).toHaveBeenCalledTimes(1);
    expect(mocks.invalidateQueries).not.toHaveBeenCalled();
  });

  it("renders the setup hero only when no portal exists", () => {
    mocks.portalState = { status: "notConfigured" };

    renderPage();

    expect(screen.getByTestId("setup-hero")).toBeDefined();
    expect(screen.queryByTestId("portal-config")).toBeNull();
  });

  it("renders the configuration view with a re-enable action when disabled", () => {
    mocks.portalState = { status: "disabled", portal };

    renderPage();

    expect(screen.queryByTestId("setup-hero")).toBeNull();
    // The whole configuration view — branding, integration docs, and the
    // danger zone — stays reachable without re-enabling first.
    expect(screen.getByTestId("portal-config")).toBeDefined();

    fireEvent.click(screen.getByRole("button", { name: "Re-enable portal" }));
    expect(mocks.updateMutation.mutate).toHaveBeenCalledWith({
      portal: "portal_123",
      enabled: true,
    });
  });

  it("hands the fetched portal to the configuration view when enabled", () => {
    mocks.portalState = { status: "enabled", portal: { ...portal, enabled: true } };

    renderPage();

    expect(screen.queryByText("Portal disabled")).toBeNull();
    // Disabling now lives inside the configuration view, which needs the row
    // itself rather than a callback.
    expect(screen.getByTestId("portal-config").textContent).toBe("acme|ks_123");
  });
});
