import { type PlanLimits, freeTierLimits, limitsByPlan } from "@/lib/limits";
import { DEPLOY_PLANS, type DeployPlan } from "@/lib/stripe/deployPlan";

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

export type SizeUnlock = { type: "plan"; plan: DeployPlan } | { type: "contact" };

export type SizeOption =
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
