import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { KeyDetailsDrawer } from "./key-details-drawer";

const useFetchRequestDetails = vi.fn();

vi.mock("../hooks/use-fetch-request-details", () => ({
  useFetchRequestDetails: (args: { requestId?: string }) => useFetchRequestDetails(args),
}));

vi.mock("@unkey/ui", () => ({
  Badge: ({ children, className }: { children: React.ReactNode; className?: string }) => (
    <span className={className}>{children}</span>
  ),
  Button: ({
    children,
    onClick,
    "aria-label": ariaLabel,
  }: {
    children: React.ReactNode;
    onClick?: () => void;
    "aria-label"?: string;
  }) => (
    <button type="button" onClick={onClick} aria-label={ariaLabel}>
      {children}
    </button>
  ),
  CopyButton: () => null,
  TimestampInfo: ({ value }: { value: number }) => <span>{value}</span>,
  InfoTooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  toast: { success: vi.fn(), error: vi.fn() },
}));

vi.mock("@unkey/icons", () => ({
  XMark: () => <span>close</span>,
}));

const selectedLog: KeyDetailsLog = {
  request_id: "req_123",
  time: 1_700_000_000_000,
  region: "us-east-1",
  outcome: "VALID",
  tags: ["prod"],
};

describe("KeyDetailsDrawer", () => {
  beforeEach(() => {
    useFetchRequestDetails.mockReset();
  });

  afterEach(() => {
    cleanup();
  });

  it("renders verification details when the HTTP request log is missing", () => {
    useFetchRequestDetails.mockReturnValue({
      log: undefined,
      isLoading: false,
      error: null,
    });

    render(
      <KeyDetailsDrawer
        distanceToTop={0}
        selectedLog={selectedLog}
        onLogSelect={vi.fn()}
        keyId="key_123"
        keyspaceId="ks_123"
        apiId="api_123"
      />,
    );

    expect(screen.getAllByText("VALID").length).toBeGreaterThan(0);
    expect(screen.getAllByText("req_123").length).toBeGreaterThan(0);
    expect(screen.getByText("key_123")).not.toBeNull();
    expect(screen.getByText("ks_123")).not.toBeNull();
    expect(screen.getByText("api_123")).not.toBeNull();
    expect(screen.getByText("us-east-1")).not.toBeNull();
    expect(
      screen.getByText("No request headers or body were logged for this verification."),
    ).not.toBeNull();
  });

  it("renders HTTP request details when the request log exists", () => {
    useFetchRequestDetails.mockReturnValue({
      log: {
        request_id: "req_123",
        time: 1_700_000_000_000,
        workspace_id: "ws_123",
        host: "api.unkey.com",
        method: "POST",
        path: "/v2/keys.verifyKey",
        request_headers: ["content-type: application/json"],
        request_body: "{}",
        response_status: 200,
        response_headers: ["content-type: application/json"],
        response_body: '{"data":{"code":"VALID","permissions":[]}}',
        error: "",
        service_latency: 12,
      },
      isLoading: false,
      error: null,
    });

    render(
      <KeyDetailsDrawer
        distanceToTop={0}
        selectedLog={selectedLog}
        onLogSelect={vi.fn()}
        keyId="key_123"
        keyspaceId="ks_123"
        apiId="api_123"
      />,
    );

    expect(screen.getByText("POST")).not.toBeNull();
    expect(screen.getAllByText("/v2/keys.verifyKey").length).toBeGreaterThan(0);
    expect(screen.getByText("Request Header")).not.toBeNull();
    expect(
      screen.queryByText("No request headers or body were logged for this verification."),
    ).toBeNull();
  });

  it("closes the drawer from the fallback header", () => {
    useFetchRequestDetails.mockReturnValue({
      log: undefined,
      isLoading: false,
      error: null,
    });
    const onLogSelect = vi.fn();

    render(
      <KeyDetailsDrawer
        distanceToTop={0}
        selectedLog={selectedLog}
        onLogSelect={onLogSelect}
        keyId="key_123"
        keyspaceId="ks_123"
        apiId="api_123"
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    expect(onLogSelect).toHaveBeenCalledWith(null);
  });
});
