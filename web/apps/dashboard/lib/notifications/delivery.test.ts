import { describe, expect, it } from "vitest";
import { deliveryLabels, deliveryPending } from "./delivery";

describe("test delivery status", () => {
  it("polls queued and sending deliveries", () => {
    expect(deliveryPending("pending")).toBe(true);
    expect(deliveryPending("sending")).toBe(true);
  });
  it("does not describe provider acceptance as delivery", () => {
    expect(deliveryLabels.accepted).toContain("Delivery is not confirmed");
    expect(deliveryLabels.delivered).toBe("Test delivered");
  });
  it.each(["accepted", "delivered", "failed", "suppressed", "unknown"] as const)(
    "stops polling after %s",
    (status) => {
      expect(deliveryPending(status)).toBe(false);
    },
  );
});
