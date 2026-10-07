import { limitsByPlan } from "@/lib/limits";
import { describe, expect, it } from "vitest";
import {
  PRESETS,
  formatUnit,
  planForPreset,
  presetFits,
  presetFor,
  resolveLimits,
  sizeChoice,
  sizeLabel,
  sizeOptions,
  unitFields,
  unitOptions,
} from "./sizing";

describe("unitOptions", () => {
  const fields = unitFields(resolveLimits(limitsByPlan.starter));

  it("lists every value the API accepts up to the plan limit", () => {
    expect(unitOptions(fields.cpu, 1000)).toEqual([250, 500, 750, 1000, 1250, 1500, 1750, 2000]);
    expect(unitOptions(fields.memory, 256)).toEqual([256, 512, 768, 1024, 1280, 1536, 1792, 2048]);
    expect(unitOptions(fields.storage, null).at(0)).toBe(512);
    expect(unitOptions(fields.storage, null).at(-1)).toBe(10240);
  });

  it("keeps a saved value that is off the list so it stays selectable", () => {
    expect(unitOptions(fields.cpu, 4000).at(-1)).toBe(4000);
    expect(unitOptions(fields.cpu, 1100)).toContain(1100);
  });
});

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

describe("sizeChoice", () => {
  it("picks the preset in preset mode and custom otherwise", () => {
    expect(sizeChoice({ sizeMode: "preset", cpuMillicores: 500, memoryMib: 1024 })).toEqual({
      type: "preset",
      preset: { id: "s", label: "S", cpuMillicores: 500, memoryMib: 1024 },
    });
    expect(sizeChoice({ sizeMode: "custom", cpuMillicores: 500, memoryMib: 1024 })).toEqual({
      type: "custom",
    });
    expect(sizeChoice({ sizeMode: "preset", cpuMillicores: 750, memoryMib: 1024 })).toEqual({
      type: "custom",
    });
  });

  it("labels a size by its values so the card header and the map agree", () => {
    expect(sizeLabel({ cpuMillicores: 500, memoryMib: 1024 })).toBe("S");
    expect(sizeLabel({ cpuMillicores: 750, memoryMib: 1024 })).toBe("Custom");
  });
});

describe("formatUnit", () => {
  it("writes a select value in the field's unit with up to two decimals", () => {
    const fields = unitFields(resolveLimits(limitsByPlan.pro));
    expect(formatUnit(250, fields.cpu)).toBe("0.25 vCPU");
    expect(formatUnit(1280, fields.memory)).toBe("1.25 GiB");
    expect(formatUnit(10240, fields.storage)).toBe("10 GiB");
  });
});
