import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { type Mode, type Size, onStorageTable, presetFor } from "@/lib/compute/sizing";
import { match } from "@unkey/match";

export type RegionDraft = { name: string; replicasMin: number; replicasMax: number };

export type ComputeDraft = {
  cpuMillicores: number;
  memoryMib: number;
  storageMib: number;
  regions: RegionDraft[];
};

type Range = { min: number; max: number };

export type Replicas =
  | { kind: "uniform"; min: number; max: number }
  | { kind: "mixed"; first: Range };

export type CardSlot = { kind: "saved"; name: string } | { kind: "new" };

export type SharedEdit = {
  sizeMode?: Mode;
  size?: Size;
  storageMode?: Mode;
  storageMib?: number;
  replicas?: Range;
};

export type CardEdit = SharedEdit & { region?: string };

export type EditResult =
  | { ok: true; draft: ComputeDraft; regionMissing: boolean }
  | { ok: false; reason: "unpicked" | "taken" };

export type CardView = {
  region: string | null;
  sizeMode: Mode;
  cpuMillicores: number;
  memoryMib: number;
  storageMode: Mode;
  storageMib: number;
  replicas: Replicas;
};

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

export function sameSharedEdit(a: SharedEdit, b: SharedEdit): boolean {
  return (
    a.sizeMode === b.sizeMode &&
    a.size?.cpuMillicores === b.size?.cpuMillicores &&
    a.size?.memoryMib === b.size?.memoryMib &&
    a.storageMode === b.storageMode &&
    a.storageMib === b.storageMib &&
    a.replicas?.min === b.replicas?.min &&
    a.replicas?.max === b.replicas?.max
  );
}

export function sharedPart({ region: _, ...shared }: CardEdit): SharedEdit {
  return shared;
}

export function modesOf({ sizeMode, storageMode }: SharedEdit): SharedEdit {
  return { sizeMode, storageMode };
}

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
    : { kind: "mixed", first: { min: first.replicasMin, max: first.replicasMax } };
}

function withRange(regions: RegionDraft[], range: Range): RegionDraft[] {
  return regions.map((r) => ({ ...r, replicasMin: range.min, replicasMax: range.max }));
}

function firstRange(regions: RegionDraft[]): Range {
  const first = regions.at(0);
  return { min: first?.replicasMin ?? 1, max: first?.replicasMax ?? 1 };
}

function sameNames(a: RegionDraft[], b: RegionDraft[]): boolean {
  return a.length === b.length && a.every((r, i) => r.name === b[i]?.name);
}

/** The API takes one replica range for every region, so any write that changes the regions sends a uniform range. */
function withRegions(base: ComputeDraft, regions: RegionDraft[], range?: Range): ComputeDraft {
  if (range) {
    return { ...base, regions: withRange(regions, range) };
  }
  if (sameNames(regions, base.regions)) {
    return { ...base, regions };
  }
  return { ...base, regions: withRange(regions, firstRange(regions)) };
}

export function removeRegion(base: ComputeDraft, name: string): ComputeDraft | null {
  const regions = base.regions.filter((r) => r.name !== name);
  if (regions.length === 0 || regions.length === base.regions.length) {
    return null;
  }
  return withRegions(base, regions);
}

function slotRegion(slot: CardSlot): string | null {
  return slot.kind === "saved" ? slot.name : null;
}

function overlay(base: ComputeDraft, slot: CardSlot, edit: CardEdit) {
  return {
    region: edit.region ?? slotRegion(slot),
    cpuMillicores: edit.size?.cpuMillicores ?? base.cpuMillicores,
    memoryMib: edit.size?.memoryMib ?? base.memoryMib,
    storageMib: edit.storageMib ?? base.storageMib,
  };
}

export function applyCardEdit(base: ComputeDraft, slot: CardSlot, edit: CardEdit): EditResult {
  const { region: name, ...size } = overlay(base, slot, edit);
  if (name === null) {
    return { ok: false, reason: "unpicked" };
  }
  if (name !== slotRegion(slot) && base.regions.some((r) => r.name === name)) {
    return { ok: false, reason: "taken" };
  }
  const regions = match(slot)
    .with({ kind: "saved" }, ({ name: saved }) =>
      base.regions.some((r) => r.name === saved)
        ? base.regions.map((r) => (r.name === saved ? { ...r, name } : r))
        : null,
    )
    .with({ kind: "new" }, () => [...base.regions, { name, replicasMin: 1, replicasMax: 1 }])
    .exhaustive();
  return {
    ok: true,
    draft: withRegions({ ...size, regions: base.regions }, regions ?? base.regions, edit.replicas),
    regionMissing: regions === null,
  };
}

function resolveMode(picked: Mode | undefined, onTable: boolean): Mode {
  return onTable && picked !== "custom" ? "preset" : "custom";
}

export function cardView(base: ComputeDraft, slot: CardSlot, edit: CardEdit): CardView {
  const { region, cpuMillicores, memoryMib, storageMib } = overlay(base, slot, edit);
  return {
    region,
    sizeMode: resolveMode(edit.sizeMode, presetFor(cpuMillicores, memoryMib) !== undefined),
    cpuMillicores,
    memoryMib,
    storageMode: resolveMode(edit.storageMode, onStorageTable(storageMib)),
    storageMib,
    replicas: edit.replicas
      ? { kind: "uniform", min: edit.replicas.min, max: edit.replicas.max }
      : replicasOf(base),
  };
}
