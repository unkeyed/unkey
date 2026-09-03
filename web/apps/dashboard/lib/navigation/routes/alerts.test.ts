import { describe, expect, it } from "vitest";
import { routes } from "./index";

describe("alert-scoped paths", () => {
  it("builds list and detail paths", () => {
    expect(routes.alerts.list({ workspaceSlug: "acme" })).toBe("/acme/alerts");
    expect(routes.alerts.detail({ workspaceSlug: "acme", alertId: "alert_123" })).toBe(
      "/acme/alerts/alert_123",
    );
  });
});
