import { describe, expect, it } from "vitest";
import { planFeatures } from "./plan-features";

describe("planFeatures", () => {
  it("lists starter's limits and strikes the features it lacks", () => {
    expect(planFeatures("starter")).toEqual([
      { kind: "team", label: "No team members", included: false },
      { kind: "cpu", label: "2 vCPU per instance", included: true },
      { kind: "memory", label: "2 GiB memory per instance", included: true },
      { kind: "domains", label: "1 custom domain", included: true },
      { kind: "autoscale", label: "Up to 4 instances per region", included: true },
      { kind: "logs", label: "3-day log retention", included: true },
    ]);
  });

  it("includes team members and unlimited domains on business", () => {
    expect(planFeatures("business").map((row) => row.label)).toEqual([
      "Unlimited team members",
      "16 vCPU per instance",
      "32 GiB memory per instance",
      "Unlimited custom domains",
      "Up to 16 instances per region",
      "14-day log retention",
    ]);
  });

  it("keeps rows in the same order on every plan", () => {
    for (const plan of ["starter", "pro", "business"] as const) {
      expect(planFeatures(plan).map((row) => row.kind)).toEqual([
        "team",
        "cpu",
        "memory",
        "domains",
        "autoscale",
        "logs",
      ]);
    }
  });
});
