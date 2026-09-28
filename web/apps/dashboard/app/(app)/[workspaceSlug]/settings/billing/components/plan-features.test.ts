import { describe, expect, it } from "vitest";
import { computePlanFeatures } from "./plan-features";

describe("computePlanFeatures", () => {
  it("marks team members and custom domains missing on starter", () => {
    const features = computePlanFeatures("starter");

    expect(features).toContainEqual({ kind: "team", label: "No team members", included: false });
    expect(features).toContainEqual({ kind: "domains", label: "1 custom domain", included: true });
  });

  it("includes team members and unlimited domains on pro and business", () => {
    for (const plan of ["pro", "business"] as const) {
      const features = computePlanFeatures(plan);

      expect(features).toContainEqual({
        kind: "team",
        label: "Unlimited team members",
        included: true,
      });
      expect(features).toContainEqual({
        kind: "domains",
        label: "Unlimited custom domains",
        included: true,
      });
    }
  });

  it("reads resource ceilings from the plan limits", () => {
    expect(computePlanFeatures("business").map((f) => f.label)).toEqual([
      "Unlimited team members",
      "16 vCPU per instance",
      "32 GiB memory per instance",
      "Unlimited custom domains",
      "Autoscale to 16 instances",
      "14-day log retention",
    ]);
  });
});
