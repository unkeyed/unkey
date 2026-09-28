import { describe, expect, it } from "vitest";
import { computePlanFeatures } from "./plan-features";

const labels = (plan: "starter" | "pro" | "business") =>
  computePlanFeatures(plan).features.map((feature) => feature.label);

describe("computePlanFeatures", () => {
  it("lists the base deploy features and limits on starter", () => {
    expect(computePlanFeatures("starter").inheritsFrom).toBeNull();
    expect(labels("starter")).toEqual([
      "Git push to deploy",
      "Preview deploy per PR",
      "Instant rollback",
      "2 vCPU per instance",
      "2 GiB memory per instance",
      "1 custom domain",
      "Up to 4 instances per region",
      "3-day log retention",
    ]);
  });

  it("lists only what pro adds over starter", () => {
    expect(computePlanFeatures("pro").inheritsFrom).toBe("starter");
    expect(labels("pro")).toEqual([
      "Unlimited team members",
      "8 vCPU per instance",
      "8 GiB memory per instance",
      "Unlimited custom domains",
      "Up to 8 instances per region",
      "7-day log retention",
    ]);
  });

  it("drops features business shares with pro", () => {
    expect(computePlanFeatures("business").inheritsFrom).toBe("pro");
    expect(labels("business")).toEqual([
      "16 vCPU per instance",
      "32 GiB memory per instance",
      "Up to 16 instances per region",
      "14-day log retention",
    ]);
  });
});
