import { describe, expect, it } from "vitest";
import { routes } from "./index";

describe("alert-scoped paths", () => {
  it("builds the workspace inbox path", () => {
    expect(routes.alerts.list({ workspaceSlug: "acme" })).toBe("/acme/alerts");
  });

  it("builds an email deep-link path", () => {
    expect(routes.alerts.detail({ workspaceSlug: "acme", alertId: "alert_123" })).toBe(
      "/acme/alerts/alert_123",
    );
  });
});
