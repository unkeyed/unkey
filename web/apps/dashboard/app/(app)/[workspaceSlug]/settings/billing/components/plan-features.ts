import { CUSTOM_DOMAINS_UNLIMITED, limitsByPlan } from "@/lib/limits";
import type { DeployPlan } from "@/lib/stripe/deployPlan";

export type PlanFeatureKind = "team" | "cpu" | "memory" | "domains" | "autoscale" | "logs";

export type PlanFeature = {
  kind: PlanFeatureKind;
  label: string;
  included: boolean;
};

const MIB_PER_GIB = 1024;

function customDomains(max: number): PlanFeature {
  if (max >= CUSTOM_DOMAINS_UNLIMITED) {
    return { kind: "domains", label: "Unlimited custom domains", included: true };
  }
  if (max === 0) {
    return { kind: "domains", label: "No custom domains", included: false };
  }
  return { kind: "domains", label: `${max} custom domain${max === 1 ? "" : "s"}`, included: true };
}

export function computePlanFeatures(plan: DeployPlan): PlanFeature[] {
  const limits = limitsByPlan[plan];
  return [
    limits.teamEnabled
      ? { kind: "team", label: "Unlimited team members", included: true }
      : { kind: "team", label: "No team members", included: false },
    { kind: "cpu", label: `${limits.cpuCoresMaxPerInstance} vCPU per instance`, included: true },
    {
      kind: "memory",
      label: `${limits.memoryMibMaxPerInstance / MIB_PER_GIB} GiB memory per instance`,
      included: true,
    },
    customDomains(limits.customDomainsMax),
    {
      kind: "autoscale",
      label: `Autoscale to ${limits.autoscalingReplicasMax} instances`,
      included: true,
    },
    { kind: "logs", label: `${limits.logsRetentionDaysMax}-day log retention`, included: true },
  ];
}
