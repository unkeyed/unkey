import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { limitsByPlan } from "@/lib/limits";
import { describe, expect, it } from "vitest";
import {
  type CardEdit,
  type CardSlot,
  type ComputeDraft,
  PRESETS,
  applyCardEdit,
  applyDraft,
  cardView,
  fromSettings,
  nudgeUnit,
  parseUnit,
  presetFits,
  presetFor,
  replicasOf,
  resolveLimits,
  sameDraft,
  unitFields,
  unschedulableIn,
} from "./model";

type Region = EnvironmentSettings["regions"][number];

function settings(
  overrides: Partial<Pick<EnvironmentSettings, "cpuMillicores" | "memoryMib" | "storageMib">> & {
    regions: Region[];
  },
): EnvironmentSettings {
  return {
    environmentId: "env_1",
    projectId: "proj_1",
    appId: "app_1",
    autoDeploy: true,
    dockerfile: "",
    dockerContext: ".",
    buildCommand: "",
    watchPaths: [],
    port: 8080,
    cpuMillicores: 250,
    memoryMib: 256,
    storageMib: 0,
    command: [],
    healthcheck: null,
    shutdownSignal: "SIGTERM",
    upstreamProtocol: "http1",
    openapiSpecPath: null,
    ...overrides,
  };
}

const us = { name: "us-east-1", replicasMin: 1, replicasMax: 3 };
const eu = { name: "eu-central-1", replicasMin: 2, replicasMax: 5 };

const base: ComputeDraft = {
  cpuMillicores: 250,
  memoryMib: 256,
  storageMib: 0,
  regions: [us, eu],
};

function applied(draft: ComputeDraft, slot: CardSlot, edit: CardEdit): ComputeDraft {
  const result = applyCardEdit(draft, slot, edit);
  if (!result.ok) {
    throw new Error(`edit rejected: ${result.reason}`);
  }
  return result.draft;
}

describe("fromSettings", () => {
  it("reads size, storage and each region's range", () => {
    const draft = fromSettings(
      settings({
        cpuMillicores: 1000,
        memoryMib: 2048,
        storageMib: 1024,
        regions: [us, eu],
      }),
    );
    expect(draft).toEqual({
      cpuMillicores: 1000,
      memoryMib: 2048,
      storageMib: 1024,
      regions: [
        { name: "us-east-1", replicasMin: 1, replicasMax: 3 },
        { name: "eu-central-1", replicasMin: 2, replicasMax: 5 },
      ],
    });
  });
});

describe("replicasOf", () => {
  it("reports a shared range, mixed ranges, and one instance with no regions", () => {
    expect(replicasOf({ regions: [us, { ...eu, replicasMin: 1, replicasMax: 3 }] })).toEqual({
      kind: "uniform",
      min: 1,
      max: 3,
    });
    expect(replicasOf({ regions: [us, eu] })).toEqual({ kind: "mixed" });
    expect(replicasOf({ regions: [] })).toEqual({ kind: "uniform", min: 1, max: 1 });
  });
});

describe("applyDraft", () => {
  it("writes size, storage and regions onto the settings", () => {
    const target = settings({ regions: [us] });
    applyDraft(target, {
      cpuMillicores: 2000,
      memoryMib: 4096,
      storageMib: 5120,
      regions: [{ name: "us-west-2", replicasMin: 2, replicasMax: 4 }],
    });
    expect(target.cpuMillicores).toBe(2000);
    expect(target.memoryMib).toBe(4096);
    expect(target.storageMib).toBe(5120);
    expect(target.regions).toEqual([{ name: "us-west-2", replicasMin: 2, replicasMax: 4 }]);
  });
});

describe("applyCardEdit", () => {
  it("keeps another card's saved size when a second card saves storage", () => {
    const usCard: CardSlot = { kind: "saved", name: "us-east-1" };
    const euCard: CardSlot = { kind: "saved", name: "eu-central-1" };
    const afterA = applied(base, usCard, {
      sizeMode: "preset",
      size: { cpuMillicores: 1000, memoryMib: 2048 },
    });
    const afterB = applied(afterA, euCard, { storageMib: 5120 });
    expect(afterB).toEqual({
      cpuMillicores: 1000,
      memoryMib: 2048,
      storageMib: 5120,
      regions: [
        { name: "us-east-1", replicasMin: 1, replicasMax: 3 },
        { name: "eu-central-1", replicasMin: 2, replicasMax: 5 },
      ],
    });
  });

  it("finds a card's region by name after an earlier region is removed", () => {
    const afterRemove: ComputeDraft = { ...base, regions: [eu] };
    expect(
      applied(afterRemove, { kind: "saved", name: "eu-central-1" }, { region: "us-west-2" })
        .regions,
    ).toEqual([{ name: "us-west-2", replicasMin: 2, replicasMax: 5 }]);
  });

  it("gives a swapped-in region the swapped-out region's range when ranges are mixed", () => {
    expect(
      applied(base, { kind: "saved", name: "eu-central-1" }, { region: "ap-southeast-1" }).regions,
    ).toEqual([
      { name: "us-east-1", replicasMin: 1, replicasMax: 3 },
      { name: "ap-southeast-1", replicasMin: 2, replicasMax: 5 },
    ]);
  });

  it("rejects a swap or an add onto a region that is already in use", () => {
    expect(
      applyCardEdit(base, { kind: "saved", name: "us-east-1" }, { region: "eu-central-1" }),
    ).toEqual({ ok: false, reason: "taken" });
    expect(applyCardEdit(base, { kind: "new" }, { region: "us-east-1" })).toEqual({
      ok: false,
      reason: "taken",
    });
  });

  it("rejects an edit for a region that was removed, and a new card with no region", () => {
    expect(applyCardEdit(base, { kind: "saved", name: "us-west-2" }, { storageMib: 512 })).toEqual({
      ok: false,
      reason: "missing",
    });
    expect(applyCardEdit(base, { kind: "new" }, { storageMib: 512 })).toEqual({
      ok: false,
      reason: "unpicked",
    });
  });

  it("adds a new region with the first region's range and the card's shared settings", () => {
    expect(applied(base, { kind: "new" }, { region: "us-west-2", storageMib: 1024 })).toEqual({
      cpuMillicores: 250,
      memoryMib: 256,
      storageMib: 1024,
      regions: [
        { name: "us-east-1", replicasMin: 1, replicasMax: 3 },
        { name: "eu-central-1", replicasMin: 2, replicasMax: 5 },
        { name: "us-west-2", replicasMin: 1, replicasMax: 3 },
      ],
    });
  });

  it("applies an edited range to every region", () => {
    expect(
      applied(base, { kind: "saved", name: "us-east-1" }, { replicas: { min: 2, max: 4 } }).regions,
    ).toEqual([
      { name: "us-east-1", replicasMin: 2, replicasMax: 4 },
      { name: "eu-central-1", replicasMin: 2, replicasMax: 4 },
    ]);
  });

  it("treats an edit equal to the saved draft as no change", () => {
    const slot: CardSlot = { kind: "saved", name: "us-east-1" };
    expect(sameDraft(applied(base, slot, { storageMib: 0, sizeMode: "custom" }), base)).toBe(true);
    expect(sameDraft(applied(base, slot, { storageMib: 512 }), base)).toBe(false);
  });
});

describe("cardView", () => {
  it("overlays the card's edit on the saved draft", () => {
    expect(
      cardView(
        base,
        { kind: "saved", name: "us-east-1" },
        { size: { cpuMillicores: 500, memoryMib: 1024 } },
      ),
    ).toEqual({
      region: "us-east-1",
      sizeMode: "preset",
      cpuMillicores: 500,
      memoryMib: 1024,
      storageMode: "preset",
      storageMib: 0,
      replicas: { kind: "mixed" },
    });
  });

  it("shows custom when picked or when the value is not on the table", () => {
    const view = cardView(
      { ...base, cpuMillicores: 750, storageMib: 1024 },
      { kind: "new" },
      { storageMode: "custom", sizeMode: "preset" },
    );
    expect([view.region, view.sizeMode, view.storageMode]).toEqual([null, "custom", "custom"]);
  });
});

describe("parseUnit", () => {
  const fields = unitFields(resolveLimits(limitsByPlan.pro));

  it("accepts on-step values in display units and returns base units", () => {
    expect(parseUnit("1.25", fields.cpu)).toEqual({ ok: true, value: 1250 });
    expect(parseUnit("1.5", fields.memory)).toEqual({ ok: true, value: 1536 });
    expect(parseUnit("2.5", fields.storage)).toEqual({ ok: true, value: 2560 });
  });

  it("rejects values the API would refuse", () => {
    expect(parseUnit("1.3", fields.cpu)).toEqual({ ok: false, message: "Use steps of 0.25 vCPU." });
    expect(parseUnit("1.1", fields.memory)).toEqual({
      ok: false,
      message: "Use steps of 0.25 GiB.",
    });
    expect(parseUnit("0.7", fields.storage)).toEqual({
      ok: false,
      message: "Use steps of 0.5 GiB.",
    });
    expect(parseUnit("0.1", fields.cpu)).toEqual({ ok: false, message: "Minimum is 0.25 vCPU." });
    expect(parseUnit("9", fields.cpu)).toEqual({
      ok: false,
      message: "Maximum is 8 vCPU on your plan.",
    });
    expect(parseUnit("", fields.memory)).toEqual({ ok: false, message: "Enter a number." });
    expect(parseUnit("abc", fields.memory)).toEqual({ ok: false, message: "Enter a number." });
  });
});

describe("nudgeUnit", () => {
  const { cpu } = unitFields(resolveLimits(limitsByPlan.pro));

  it("steps on the grid and stays within range", () => {
    expect(nudgeUnit(1000, 1, cpu)).toBe(1250);
    expect(nudgeUnit(1100, 1, cpu)).toBe(1250);
    expect(nudgeUnit(250, -1, cpu)).toBe(250);
    expect(nudgeUnit(8000, 1, cpu)).toBe(8000);
    expect(nudgeUnit(null, 1, cpu)).toBe(250);
  });
});

describe("unschedulableIn", () => {
  it("lists the regions that cannot be scheduled once regions have loaded", () => {
    const regions = [
      { name: "us-east-1", canSchedule: true },
      { name: "eu-central-1", canSchedule: false },
    ];
    expect(
      unschedulableIn({ status: "ready", regions }, ["us-east-1", "eu-central-1", "local"]),
    ).toEqual(["eu-central-1"]);
    expect(unschedulableIn({ status: "loading" }, ["eu-central-1"])).toEqual([]);
  });
});

describe("presets", () => {
  it("finds the preset for an exact size only", () => {
    expect(presetFor(2000, 4096)?.label).toBe("L");
    expect(presetFor(2000, 2048)).toBeUndefined();
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
