import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { LogFooter } from "./log-footer";

vi.mock("@unkey/ui", () => ({
  Badge: ({ children }: { children: React.ReactNode }) => <span>{children}</span>,
  TimestampInfo: ({ value }: { value: number }) => <span>{value}</span>,
  InfoTooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  toast: { success: vi.fn(), error: vi.fn() },
}));

describe("LogFooter", () => {
  it("renders without crashing when the response body has no permissions", () => {
    render(
      <LogFooter
        log={{
          request_id: "req_123",
          time: 1_700_000_000_000,
          workspace_id: "ws_123",
          host: "api.example.com",
          method: "POST",
          path: "/v2/keys.verifyKey",
          request_headers: [],
          request_body: "",
          response_status: 200,
          response_headers: [],
          response_body: "",
          error: "",
          service_latency: 1,
        }}
      />,
    );

    expect(screen.getByText("req_123")).not.toBeNull();
    expect(screen.getByText("api.example.com")).not.toBeNull();
    expect(screen.queryByText("Permissions")).toBeNull();
  });
});
