import { limitsByPlan } from "@/lib/limits";
import { describe, expect, it } from "vitest";
import { PRESETS, presetFits, resolveLimits, sizeOptions, unitFields, unitOptions } from "./sizing";

describe("unitOptions", () => {
  const fields = unitFields(resolveLimits(limitsByPlan.starter));

  it("keeps a saved value that is off the list so it stays selectable", () => {
    expect(unitOptions(fields.cpu, 4000).at(-1)).toBe(4000);
    expect(unitOptions(fields.cpu, 1100)).toContain(1100);
  });
});

describe("presets", () => {
  it("locks presets above the plan limit", () => {
    const limits = resolveLimits(limitsByPlan.starter);
    const fitting = PRESETS.filter((p) => presetFits(p, limits)).map((p) => p.label);
    expect(fitting).toEqual(["XS", "S", "M"]);
  });
});

describe("sizeOptions", () => {
  const summary = (plan: keyof typeof limitsByPlan) =>
    sizeOptions(resolveLimits(limitsByPlan[plan])).map((option) =>
      option.type === "available"
        ? option.preset.label
        : `${option.preset.label}:${option.unlock.type === "plan" ? option.unlock.plan : "contact"}`,
    );

  it("locks sizes above the plan and names the plan that unlocks them", () => {
    expect(summary("free")).toEqual([
      "XS",
      "S",
      "M",
      "L",
      "XL:pro",
      "2XL:business",
      "4XL:business",
    ]);
    expect(summary("pro")).toEqual(["XS", "S", "M", "L", "XL", "2XL:business", "4XL:business"]);
    expect(summary("business")).toEqual(["XS", "S", "M", "L", "XL", "2XL", "4XL"]);
  });
});
