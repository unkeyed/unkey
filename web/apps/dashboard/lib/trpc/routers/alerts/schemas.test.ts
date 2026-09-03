import { describe, expect, it } from "vitest";
import { listAlertsInput, resolveAlertInput } from "./schemas";

describe("alert inputs", () => {
  it("applies list defaults", () => {
    expect(listAlertsInput.parse({})).toEqual({ status: "open", limit: 50 });
  });

  it("accepts traffic drop alerts", () => {
    expect(listAlertsInput.parse({ metric: "requests_drop" })).toEqual({
      status: "open",
      metric: "requests_drop",
      limit: 50,
    });
  });

  it.each([
    { status: "active" },
    { metric: "latency" },
    { limit: 0 },
    { limit: 101 },
    { cursor: "" },
  ])("rejects invalid list input", (input) => {
    expect(listAlertsInput.safeParse(input).success).toBe(false);
  });

  it("trims a resolution message", () => {
    expect(resolveAlertInput.parse({ alertId: "alert_1", message: "  Fixed capacity  " })).toEqual({
      alertId: "alert_1",
      message: "Fixed capacity",
    });
  });

  it.each(["", " ", "a".repeat(1001)])("rejects invalid resolution message", (message) => {
    expect(resolveAlertInput.safeParse({ alertId: "alert_1", message }).success).toBe(false);
  });
});
