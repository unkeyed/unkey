import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { limitsByPlan } from "@/lib/limits";
import { describe, expect, it } from "vitest";
import {
  type ComputeDraft,
  PRESETS,
  applyDraft,
  diffDraft,
  fromSettings,
  presetFits,
  presetFor,
  resolveLimits,
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

const base: ComputeDraft = {
  sizeMode: "preset",
  cpuMillicores: 250,
  memoryMib: 256,
  storageMib: 0,
  replicas: { kind: "uniform", min: 1, max: 1 },
  regions: ["us-east-1"],
};

describe("fromSettings", () => {
  it("reads a shared range and a preset size", () => {
    const draft = fromSettings(
      settings({
        cpuMillicores: 1000,
        memoryMib: 2048,
        storageMib: 1024,
        regions: [
          { name: "us-east-1", replicasMin: 1, replicasMax: 3 },
          { name: "local", replicasMin: 1, replicasMax: 3 },
        ],
      }),
    );
    expect(draft).toEqual({
      sizeMode: "preset",
      cpuMillicores: 1000,
      memoryMib: 2048,
      storageMib: 1024,
      replicas: { kind: "uniform", min: 1, max: 3 },
      regions: ["us-east-1", "local"],
    });
  });

  it("marks differing ranges as mixed and an off-table size as custom", () => {
    const draft = fromSettings(
      settings({
        cpuMillicores: 750,
        memoryMib: 1536,
        regions: [
          { name: "us-east-1", replicasMin: 1, replicasMax: 3 },
          { name: "eu-central-1", replicasMin: 2, replicasMax: 4 },
        ],
      }),
    );
    expect(draft.replicas).toEqual({ kind: "mixed" });
    expect(draft.sizeMode).toBe("custom");
  });

  it("defaults to one instance when there are no regions", () => {
    const draft = fromSettings(settings({ regions: [] }));
    expect(draft.replicas).toEqual({ kind: "uniform", min: 1, max: 1 });
    expect(draft.regions).toEqual([]);
  });
});

describe("applyDraft", () => {
  it("writes a uniform range to every region", () => {
    const target = settings({
      regions: [
        { name: "us-east-1", replicasMin: 1, replicasMax: 1 },
        { name: "eu-central-1", replicasMin: 2, replicasMax: 5 },
      ],
    });
    applyDraft(target, {
      ...base,
      cpuMillicores: 2000,
      memoryMib: 4096,
      storageMib: 5120,
      replicas: { kind: "uniform", min: 2, max: 4 },
      regions: ["us-east-1", "eu-central-1"],
    });
    expect(target.cpuMillicores).toBe(2000);
    expect(target.memoryMib).toBe(4096);
    expect(target.storageMib).toBe(5120);
    expect(target.regions).toEqual([
      { name: "us-east-1", replicasMin: 2, replicasMax: 4 },
      { name: "eu-central-1", replicasMin: 2, replicasMax: 4 },
    ]);
  });

  it("keeps per-region ranges when mixed and gives new regions the first range", () => {
    const target = settings({
      regions: [
        { name: "us-east-1", replicasMin: 1, replicasMax: 3 },
        { name: "eu-central-1", replicasMin: 2, replicasMax: 5 },
      ],
    });
    applyDraft(target, {
      ...base,
      replicas: { kind: "mixed" },
      regions: ["us-east-1", "eu-central-1", "ap-southeast-1"],
    });
    expect(target.regions).toEqual([
      { name: "us-east-1", replicasMin: 1, replicasMax: 3 },
      { name: "eu-central-1", replicasMin: 2, replicasMax: 5 },
      { name: "ap-southeast-1", replicasMin: 1, replicasMax: 3 },
    ]);
  });

  it("swaps a region in place", () => {
    const target = settings({
      regions: [
        { name: "us-east-1", replicasMin: 1, replicasMax: 2 },
        { name: "eu-central-1", replicasMin: 1, replicasMax: 2 },
      ],
    });
    applyDraft(target, {
      ...base,
      replicas: { kind: "uniform", min: 1, max: 2 },
      regions: ["us-west-2", "eu-central-1"],
    });
    expect(target.regions).toEqual([
      { name: "us-west-2", replicasMin: 1, replicasMax: 2 },
      { name: "eu-central-1", replicasMin: 1, replicasMax: 2 },
    ]);
  });
});

describe("diffDraft", () => {
  it("lists every change", () => {
    const next: ComputeDraft = {
      sizeMode: "preset",
      cpuMillicores: 500,
      memoryMib: 1024,
      storageMib: 1024,
      replicas: { kind: "uniform", min: 1, max: 3 },
      regions: ["eu-central-1"],
    };
    expect(diffDraft(base, next)).toEqual([
      "Size XS (1/4 vCPU · 256 MiB) → S (1/2 vCPU · 1 GiB)",
      "Storage None → 1 GiB",
      "Instances 1 → 1–3",
      "+ eu-central-1",
      "− us-east-1",
    ]);
  });

  it("labels a custom size and mixed ranges", () => {
    const next: ComputeDraft = {
      ...base,
      sizeMode: "custom",
      cpuMillicores: 750,
      replicas: { kind: "mixed" },
    };
    expect(diffDraft(base, next)).toEqual([
      "Size XS (1/4 vCPU · 256 MiB) → Custom (3/4 vCPU · 256 MiB)",
      "Instances 1 → Mixed",
    ]);
  });

  it("reports nothing when the drafts match", () => {
    expect(diffDraft(base, { ...base, regions: ["us-east-1"] })).toEqual([]);
    expect(
      diffDraft({ ...base, replicas: { kind: "mixed" } }, { ...base, replicas: { kind: "mixed" } }),
    ).toEqual([]);
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
