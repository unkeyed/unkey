import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { type PlanLimits, freeTierLimits } from "@/lib/limits";
import {
  formatCpuParts,
  formatMemoryParts,
  formatStorageParts,
} from "@/lib/utils/deployment-formatters";
import { match } from "@unkey/match";

export type Preset = { id: string; label: string; cpuMillicores: number; memoryMib: number };

export const PRESETS: readonly Preset[] = [
  { id: "xs", label: "XS", cpuMillicores: 250, memoryMib: 256 },
  { id: "s", label: "S", cpuMillicores: 500, memoryMib: 1024 },
  { id: "m", label: "M", cpuMillicores: 1000, memoryMib: 2048 },
  { id: "l", label: "L", cpuMillicores: 2000, memoryMib: 4096 },
  { id: "xl", label: "XL", cpuMillicores: 4000, memoryMib: 8192 },
  { id: "2xl", label: "2XL", cpuMillicores: 8000, memoryMib: 16384 },
  { id: "4xl", label: "4XL", cpuMillicores: 16000, memoryMib: 32768 },
];

export const STORAGE_OPTIONS = [0, 512, 1024, 2048, 5120, 10240, 20480, 51200] as const;

export type RegionFlag = "us" | "de" | "sg";

export type RegionInfo = {
  name: string;
  city: string;
  flag: RegionFlag | null;
  pin: { lon: number; lat: number; label: "left" | "right" } | null;
};

const REGIONS = new Map<string, Omit<RegionInfo, "name">>([
  ["us-west-2", { city: "Oregon", flag: "us", pin: { lon: -120.5, lat: 45.8, label: "left" } }],
  [
    "us-east-1",
    { city: "N. Virginia", flag: "us", pin: { lon: -77.5, lat: 38.9, label: "right" } },
  ],
  ["eu-central-1", { city: "Frankfurt", flag: "de", pin: { lon: 8.7, lat: 50.1, label: "right" } }],
  [
    "ap-southeast-1",
    { city: "Singapore", flag: "sg", pin: { lon: 103.8, lat: 1.35, label: "left" } },
  ],
]);

export function regionInfo(name: string): RegionInfo {
  return { name, ...(REGIONS.get(name) ?? { city: name, flag: null, pin: null }) };
}

export type Replicas = { kind: "uniform"; min: number; max: number } | { kind: "mixed" };

export type ComputeDraft = {
  sizeMode: "preset" | "custom";
  cpuMillicores: number;
  memoryMib: number;
  storageMib: number;
  replicas: Replicas;
  regions: string[];
};

export type ComputeLimits = {
  cpuMillicores: number;
  memoryMib: number;
  storageMib: number;
  replicas: number;
};

type LimitSource = Pick<
  PlanLimits,
  | "cpuCoresMaxPerInstance"
  | "memoryMibMaxPerInstance"
  | "storageMibMaxPerInstance"
  | "autoscalingReplicasMax"
>;

export function resolveLimits(limits: LimitSource | null): ComputeLimits {
  const source = limits ?? freeTierLimits;
  return {
    cpuMillicores: source.cpuCoresMaxPerInstance * 1000,
    memoryMib: source.memoryMibMaxPerInstance,
    storageMib: source.storageMibMaxPerInstance,
    replicas: Math.max(1, source.autoscalingReplicasMax),
  };
}

export function presetFor(cpuMillicores: number, memoryMib: number): Preset | undefined {
  return PRESETS.find((p) => p.cpuMillicores === cpuMillicores && p.memoryMib === memoryMib);
}

export function presetFits(preset: Preset, limits: ComputeLimits): boolean {
  return preset.cpuMillicores <= limits.cpuMillicores && preset.memoryMib <= limits.memoryMib;
}

type Size = Pick<ComputeDraft, "sizeMode" | "cpuMillicores" | "memoryMib">;

export function activePreset(size: Size): Preset | undefined {
  return match(size.sizeMode)
    .with("custom", () => undefined)
    .with("preset", () => presetFor(size.cpuMillicores, size.memoryMib))
    .exhaustive();
}

export function fromSettings(settings: EnvironmentSettings): ComputeDraft {
  const [first, ...rest] = settings.regions;
  let replicas: Replicas = { kind: "uniform", min: 1, max: 1 };
  if (first) {
    const same = rest.every(
      (r) => r.replicasMin === first.replicasMin && r.replicasMax === first.replicasMax,
    );
    replicas = same
      ? { kind: "uniform", min: first.replicasMin, max: first.replicasMax }
      : { kind: "mixed" };
  }
  return {
    sizeMode: presetFor(settings.cpuMillicores, settings.memoryMib) ? "preset" : "custom",
    cpuMillicores: settings.cpuMillicores,
    memoryMib: settings.memoryMib,
    storageMib: settings.storageMib,
    replicas,
    regions: settings.regions.map((r) => r.name),
  };
}

export function applyDraft(target: EnvironmentSettings, draft: ComputeDraft): void {
  target.cpuMillicores = draft.cpuMillicores;
  target.memoryMib = draft.memoryMib;
  target.storageMib = draft.storageMib;
  const first = target.regions.at(0) ?? { replicasMin: 1, replicasMax: 1 };
  const existing = new Map(target.regions.map((r) => [r.name, r]));
  const { replicas } = draft;
  target.regions = draft.regions.map((name) =>
    match(replicas)
      .with({ kind: "uniform" }, ({ min, max }) => ({ name, replicasMin: min, replicasMax: max }))
      .with(
        { kind: "mixed" },
        () =>
          existing.get(name) ?? {
            name,
            replicasMin: first.replicasMin,
            replicasMax: first.replicasMax,
          },
      )
      .exhaustive(),
  );
}

const joinParts = ({ value, unit }: { value: string; unit: string }) =>
  unit ? `${value} ${unit}` : value;

export const formatCpu = (millicores: number) => joinParts(formatCpuParts(millicores));
export const formatMemory = (mib: number) => joinParts(formatMemoryParts(mib));
export const formatStorage = (mib: number) => joinParts(formatStorageParts(mib));

export function formatReplicas(replicas: Replicas): string {
  return match(replicas)
    .with({ kind: "mixed" }, () => "Mixed")
    .with({ kind: "uniform" }, ({ min, max }) => (min === max ? `${max}` : `${min}–${max}`))
    .exhaustive();
}

export function sizeLabel(size: Size): string {
  return activePreset(size)?.label ?? "Custom";
}

function sizeSpec(size: Size): string {
  return `${formatCpu(size.cpuMillicores)} · ${formatMemory(size.memoryMib)}`;
}

export function inheritedSummary(draft: ComputeDraft): string {
  const single = draft.replicas.kind === "uniform" && draft.replicas.max === 1;
  const noun = single ? "instance" : "instances";
  return `${formatReplicas(draft.replicas)} ${noun} · ${sizeLabel(draft)} · ${sizeSpec(draft)}`;
}

export function diffDraft(base: ComputeDraft, draft: ComputeDraft): string[] {
  const changes: string[] = [];
  if (base.cpuMillicores !== draft.cpuMillicores || base.memoryMib !== draft.memoryMib) {
    changes.push(
      `Size ${sizeLabel(base)} (${sizeSpec(base)}) → ${sizeLabel(draft)} (${sizeSpec(draft)})`,
    );
  }
  if (base.storageMib !== draft.storageMib) {
    changes.push(`Storage ${formatStorage(base.storageMib)} → ${formatStorage(draft.storageMib)}`);
  }
  if (formatReplicas(base.replicas) !== formatReplicas(draft.replicas)) {
    changes.push(`Instances ${formatReplicas(base.replicas)} → ${formatReplicas(draft.replicas)}`);
  }
  for (const name of draft.regions) {
    if (!base.regions.includes(name)) {
      changes.push(`+ ${name}`);
    }
  }
  for (const name of base.regions) {
    if (!draft.regions.includes(name)) {
      changes.push(`− ${name}`);
    }
  }
  return changes;
}
