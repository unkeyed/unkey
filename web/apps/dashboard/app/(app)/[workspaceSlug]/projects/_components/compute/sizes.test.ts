import { limitsByPlan } from "@/lib/limits";
import { describe, expect, it } from "vitest";
import { PRESETS, planForPreset, presetFits, presetFor, resolveLimits, sizeOptions } from "./sizes";

describe("presets", () => {
  it("finds the preset for an exact size only", () => {
    expect(presetFor(2000, 4096)?.label).toBe("L");
    expect(presetFor(2000, 2048)).toBeUndefined();
  });

  it("names the smallest plan that fits each preset", () => {
    expect(PRESETS.map((p) => [p.label, planForPreset(p)])).toEqual([
      ["XS", "starter"],
      ["S", "starter"],
      ["M", "starter"],
      ["L", "pro"],
      ["XL", "pro"],
      ["2XL", "business"],
      ["4XL", "business"],
    ]);
  });

  it("locks presets above the plan limit", () => {
    const limits = resolveLimits(limitsByPlan.starter);
    const fitting = PRESETS.filter((p) => presetFits(p, limits)).map((p) => p.label);
    expect(fitting).toEqual(["XS", "S", "M"]);
  });
});

describe("resolveLimits", () => {
  it("converts plan limits to millicores", () => {
    expect(resolveLimits(limitsByPlan.pro)).toEqual({
      cpuMillicores: 8000,
      memoryMib: 8192,
      storageMib: 10240,
      replicas: 8,
    });
  });

  it("falls back to the free tier and keeps at least one replica", () => {
    expect(resolveLimits(null)).toEqual({
      cpuMillicores: 2000,
      memoryMib: 4096,
      storageMib: 10240,
      replicas: 1,
    });
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
