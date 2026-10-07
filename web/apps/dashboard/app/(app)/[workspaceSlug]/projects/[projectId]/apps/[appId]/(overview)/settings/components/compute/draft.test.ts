import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { describe, expect, it } from "vitest";
import {
  type CardEdit,
  type CardSlot,
  type ComputeDraft,
  applyCardEdit,
  applyDraft,
  cardView,
  fromSettings,
  removeRegion,
  replicasOf,
  sameDraft,
  sameSharedEdit,
} from "./draft";

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
    expect(replicasOf({ regions: [us, eu] })).toEqual({
      kind: "mixed",
      first: { min: 1, max: 3 },
    });
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

  it("sends one range for every region when a swap changes regions with mixed ranges", () => {
    expect(
      applied(base, { kind: "saved", name: "eu-central-1" }, { region: "ap-southeast-1" }).regions,
    ).toEqual([
      { name: "us-east-1", replicasMin: 1, replicasMax: 3 },
      { name: "ap-southeast-1", replicasMin: 1, replicasMax: 3 },
    ]);
  });

  it("keeps mixed ranges when only the size changes", () => {
    expect(
      applied(base, { kind: "saved", name: "eu-central-1" }, { storageMib: 512 }).regions,
    ).toEqual([us, eu]);
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

  it("applies env-wide fields for a region that is absent, and rejects a new card with no region", () => {
    expect(
      applyCardEdit(
        base,
        { kind: "saved", name: "us-west-2" },
        { storageMib: 512, region: "ap-southeast-1" },
      ),
    ).toEqual({
      ok: true,
      draft: { ...base, storageMib: 512 },
      regionMissing: true,
    });
    expect(applyCardEdit(base, { kind: "new" }, { storageMib: 512 })).toEqual({
      ok: false,
      reason: "unpicked",
    });
  });

  it("adds a new region and gives every region the first region's range", () => {
    expect(applied(base, { kind: "new" }, { region: "us-west-2", storageMib: 1024 })).toEqual({
      cpuMillicores: 250,
      memoryMib: 256,
      storageMib: 1024,
      regions: [
        { name: "us-east-1", replicasMin: 1, replicasMax: 3 },
        { name: "eu-central-1", replicasMin: 1, replicasMax: 3 },
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
      replicas: { kind: "mixed", first: { min: 1, max: 3 } },
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

describe("removeRegion", () => {
  it("removes a region and sends one range for the rest", () => {
    const three: ComputeDraft = {
      ...base,
      regions: [us, eu, { name: "us-west-2", replicasMin: 4, replicasMax: 6 }],
    };
    expect(removeRegion(three, "us-east-1")?.regions).toEqual([
      { name: "eu-central-1", replicasMin: 2, replicasMax: 5 },
      { name: "us-west-2", replicasMin: 2, replicasMax: 5 },
    ]);
  });

  it("refuses to remove the last region or a region that is absent", () => {
    expect(removeRegion({ ...base, regions: [us] }, "us-east-1")).toBeNull();
    expect(removeRegion(base, "us-west-2")).toBeNull();
  });
});

describe("sameSharedEdit", () => {
  it("compares field by field and ignores absent fields", () => {
    expect(sameSharedEdit({ storageMib: 512, sizeMode: undefined }, { storageMib: 512 })).toBe(
      true,
    );
    expect(
      sameSharedEdit(
        { size: { cpuMillicores: 500, memoryMib: 1024 } },
        { size: { cpuMillicores: 500, memoryMib: 2048 } },
      ),
    ).toBe(false);
    expect(sameSharedEdit({ replicas: { min: 1, max: 2 } }, {})).toBe(false);
  });
});
