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

export type RegionDraft = { name: string; replicasMin: number; replicasMax: number };

export type ComputeDraft = {
  cpuMillicores: number;
  memoryMib: number;
  storageMib: number;
  regions: RegionDraft[];
};

export type Mode = "preset" | "custom";

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

type Size = Pick<ComputeDraft, "cpuMillicores" | "memoryMib">;

export function replicasOf(draft: Pick<ComputeDraft, "regions">): Replicas {
  const [first, ...rest] = draft.regions;
  if (!first) {
    return { kind: "uniform", min: 1, max: 1 };
  }
  const same = rest.every(
    (r) => r.replicasMin === first.replicasMin && r.replicasMax === first.replicasMax,
  );
  return same
    ? { kind: "uniform", min: first.replicasMin, max: first.replicasMax }
    : { kind: "mixed" };
}

export function fromSettings(settings: EnvironmentSettings): ComputeDraft {
  return {
    cpuMillicores: settings.cpuMillicores,
    memoryMib: settings.memoryMib,
    storageMib: settings.storageMib,
    regions: settings.regions.map(({ name, replicasMin, replicasMax }) => ({
      name,
      replicasMin,
      replicasMax,
    })),
  };
}

export function applyDraft(target: EnvironmentSettings, draft: ComputeDraft): void {
  target.cpuMillicores = draft.cpuMillicores;
  target.memoryMib = draft.memoryMib;
  target.storageMib = draft.storageMib;
  target.regions = draft.regions.map((r) => ({ ...r }));
}

export function sameDraft(a: ComputeDraft, b: ComputeDraft): boolean {
  return (
    a.cpuMillicores === b.cpuMillicores &&
    a.memoryMib === b.memoryMib &&
    a.storageMib === b.storageMib &&
    a.regions.length === b.regions.length &&
    a.regions.every((r, i) => {
      const other = b.regions[i];
      return (
        other !== undefined &&
        r.name === other.name &&
        r.replicasMin === other.replicasMin &&
        r.replicasMax === other.replicasMax
      );
    })
  );
}

export type CardSlot = { kind: "saved"; name: string } | { kind: "new" };

export type CardEdit = {
  region?: string;
  sizeMode?: Mode;
  size?: Size;
  storageMode?: Mode;
  storageMib?: number;
  replicas?: { min: number; max: number };
};

export type EditResult =
  | { ok: true; draft: ComputeDraft }
  | { ok: false; reason: "missing" | "unpicked" | "taken" };

export function applyCardEdit(base: ComputeDraft, slot: CardSlot, edit: CardEdit): EditResult {
  const own = slot.kind === "saved" ? slot.name : null;
  const name = edit.region ?? own;
  if (name === null) {
    return { ok: false, reason: "unpicked" };
  }
  if (name !== own && base.regions.some((r) => r.name === name)) {
    return { ok: false, reason: "taken" };
  }
  const regions = match(slot)
    .with({ kind: "saved" }, ({ name: saved }) =>
      base.regions.some((r) => r.name === saved)
        ? base.regions.map((r) => (r.name === saved ? { ...r, name } : r))
        : null,
    )
    .with({ kind: "new" }, () => {
      const first = base.regions.at(0);
      return [
        ...base.regions,
        { name, replicasMin: first?.replicasMin ?? 1, replicasMax: first?.replicasMax ?? 1 },
      ];
    })
    .exhaustive();
  if (!regions) {
    return { ok: false, reason: "missing" };
  }
  const range = edit.replicas;
  return {
    ok: true,
    draft: {
      cpuMillicores: edit.size?.cpuMillicores ?? base.cpuMillicores,
      memoryMib: edit.size?.memoryMib ?? base.memoryMib,
      storageMib: edit.storageMib ?? base.storageMib,
      regions: range
        ? regions.map((r) => ({ ...r, replicasMin: range.min, replicasMax: range.max }))
        : regions,
    },
  };
}

export type CardView = {
  region: string | null;
  sizeMode: Mode;
  cpuMillicores: number;
  memoryMib: number;
  storageMode: Mode;
  storageMib: number;
  replicas: Replicas;
};

function resolveMode(picked: Mode | undefined, onTable: boolean): Mode {
  return onTable && picked !== "custom" ? "preset" : "custom";
}

export function cardView(base: ComputeDraft, slot: CardSlot, edit: CardEdit): CardView {
  const cpuMillicores = edit.size?.cpuMillicores ?? base.cpuMillicores;
  const memoryMib = edit.size?.memoryMib ?? base.memoryMib;
  const storageMib = edit.storageMib ?? base.storageMib;
  return {
    region: edit.region ?? (slot.kind === "saved" ? slot.name : null),
    sizeMode: resolveMode(edit.sizeMode, presetFor(cpuMillicores, memoryMib) !== undefined),
    cpuMillicores,
    memoryMib,
    storageMode: resolveMode(
      edit.storageMode,
      STORAGE_OPTIONS.some((mib) => mib === storageMib),
    ),
    storageMib,
    replicas: edit.replicas
      ? { kind: "uniform", min: edit.replicas.min, max: edit.replicas.max }
      : replicasOf(base),
  };
}

export function activePreset(view: Pick<CardView, "sizeMode" | "cpuMillicores" | "memoryMib">) {
  return match(view.sizeMode)
    .with("custom", () => undefined)
    .with("preset", () => presetFor(view.cpuMillicores, view.memoryMib))
    .exhaustive();
}

export type AvailableRegion = { name: string; canSchedule: boolean };

export type AvailableRegions =
  | { status: "loading" }
  | { status: "error" }
  | { status: "ready"; regions: AvailableRegion[] };

export function unschedulableIn(available: AvailableRegions, names: string[]): string[] {
  if (available.status !== "ready") {
    return [];
  }
  const blocked = new Set(available.regions.filter((r) => !r.canSchedule).map((r) => r.name));
  return names.filter((name) => blocked.has(name));
}

export function addRegionBlocker(available: AvailableRegions, names: string[]): string | null {
  return match(available)
    .with({ status: "loading" }, () => "Loading regions…")
    .with({ status: "error" }, () => "Couldn't load regions. Reload the page to try again.")
    .with({ status: "ready" }, ({ regions }) =>
      regions.some((r) => r.canSchedule && !names.includes(r.name))
        ? null
        : "Your app already runs in every available region.",
    )
    .exhaustive();
}

export type UnitField = { unit: string; scale: number; min: number; max: number; step: number };

export type UnitParse = { ok: true; value: number } | { ok: false; message: string };

const MIB_PER_GIB = 1024;

export function unitFields(limits: ComputeLimits) {
  return {
    cpu: { unit: "vCPU", scale: 1000, min: 250, max: limits.cpuMillicores, step: 250 },
    memory: { unit: "GiB", scale: MIB_PER_GIB, min: 256, max: limits.memoryMib, step: 256 },
    storage: { unit: "GiB", scale: MIB_PER_GIB, min: 512, max: limits.storageMib, step: 512 },
  } satisfies Record<string, UnitField>;
}

export function formatUnit(value: number, field: UnitField): string {
  return String(Math.round((value / field.scale) * 100) / 100);
}

export function parseUnit(text: string, field: UnitField): UnitParse {
  const display = Number(text.trim());
  if (text.trim() === "" || !Number.isFinite(display)) {
    return { ok: false, message: "Enter a number." };
  }
  const raw = display * field.scale;
  const value = Math.round(raw);
  if (value < field.min) {
    return { ok: false, message: `Minimum is ${formatUnit(field.min, field)} ${field.unit}.` };
  }
  if (value > field.max) {
    return {
      ok: false,
      message: `Maximum is ${formatUnit(field.max, field)} ${field.unit} on your plan.`,
    };
  }
  if (Math.abs(raw - value) > 1e-6 || value % field.step !== 0) {
    return {
      ok: false,
      message: `Use steps of ${formatUnit(field.step, field)} ${field.unit}.`,
    };
  }
  return { ok: true, value };
}

export function nudgeUnit(value: number | null, direction: 1 | -1, field: UnitField): number {
  if (value === null) {
    return field.min;
  }
  const snapped = Math.round(value / field.step) * field.step;
  return Math.min(field.max, Math.max(field.min, snapped + direction * field.step));
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
  return presetFor(size.cpuMillicores, size.memoryMib)?.label ?? "Custom";
}

export function inheritedSummary(view: CardView): string {
  const single = view.replicas.kind === "uniform" && view.replicas.max === 1;
  const noun = single ? "instance" : "instances";
  const label = activePreset(view)?.label ?? "Custom";
  return `${formatReplicas(view.replicas)} ${noun} · ${label} · ${formatCpu(view.cpuMillicores)} · ${formatMemory(view.memoryMib)}`;
}
