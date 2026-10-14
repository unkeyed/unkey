import { type PlanLimits, freeTierLimits, limitsByPlan } from "@/lib/limits";
import { DEPLOY_PLANS, type DeployPlan } from "@/lib/stripe/deployPlan";
import { formatCpu, formatMemory, formatStorage } from "@/lib/utils/deployment-formatters";

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

const STORAGE_OPTIONS = [0, 512, 1024, 2048, 5120, 10240, 20480, 51200] as const;

export const CUSTOM_SIZE_LABEL = "Custom";

export type Size = { cpuMillicores: number; memoryMib: number };

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

export function planForPreset(preset: Preset): DeployPlan | undefined {
  return DEPLOY_PLANS.find((plan) => presetFits(preset, resolveLimits(limitsByPlan[plan])));
}

export function onStorageTable(storageMib: number): boolean {
  return STORAGE_OPTIONS.some((mib) => mib === storageMib);
}

export function sizeLabel(size: Size): string {
  return presetFor(size.cpuMillicores, size.memoryMib)?.label ?? CUSTOM_SIZE_LABEL;
}

type SizeChoice = { type: "preset"; preset: Preset } | { type: "custom" };

export function sizeChoice(size: Size, custom: boolean): SizeChoice {
  const preset = presetFor(size.cpuMillicores, size.memoryMib);
  return !custom && preset ? { type: "preset", preset } : { type: "custom" };
}

export type SizeUnlock = { type: "plan"; plan: DeployPlan } | { type: "contact" };

type SizeOption =
  | { type: "available"; preset: Preset }
  | { type: "locked"; preset: Preset; unlock: SizeUnlock };

export function sizeOptions(limits: ComputeLimits): SizeOption[] {
  return PRESETS.map((preset) => {
    if (presetFits(preset, limits)) {
      return { type: "available", preset };
    }
    const plan = planForPreset(preset);
    return { type: "locked", preset, unlock: plan ? { type: "plan", plan } : { type: "contact" } };
  });
}

export type UnitField = {
  label: string;
  min: number;
  max: number;
  step: number;
  format: (value: number) => string;
};

export function unitFields(limits: ComputeLimits) {
  return {
    cpu: { label: "CPU", min: 250, max: limits.cpuMillicores, step: 250, format: formatCpu },
    memory: {
      label: "Memory",
      min: 256,
      max: limits.memoryMib,
      step: 256,
      format: formatMemory,
    },
    storage: {
      label: "Storage",
      min: 512,
      max: limits.storageMib,
      step: 512,
      format: formatStorage,
    },
  } satisfies Record<string, UnitField>;
}

export function unitOptions(field: UnitField, current: number | null): number[] {
  const steps: number[] = [];
  for (let value = field.min; value <= field.max; value += field.step) {
    steps.push(value);
  }
  if (current !== null && current > 0 && !steps.includes(current)) {
    steps.push(current);
    steps.sort((x, y) => x - y);
  }
  return steps;
}

export function storagePresets(limits: ComputeLimits): number[] {
  return STORAGE_OPTIONS.filter((mib) => mib <= limits.storageMib);
}
