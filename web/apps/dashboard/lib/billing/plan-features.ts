import { CUSTOM_DOMAINS_UNLIMITED, limitsByPlan } from "@/lib/limits";
import type { DeployPlan } from "@/lib/stripe/deployPlan";

export type PlanFeatureKind = "team" | "cpu" | "memory" | "domains" | "autoscale" | "logs";

export type PlanFeature = {
  kind: PlanFeatureKind;
  label: string;
};

export const EVERY_PLAN_INCLUDES = [
  "Git push to deploy",
  "Preview deploy per PR",
  "Instant rollback",
] as const;

const MIB_PER_GIB = 1024;

function limitFeatures(plan: DeployPlan): PlanFeature[] {
  const limits = limitsByPlan[plan];
  const features: PlanFeature[] = [];
  if (limits.teamEnabled) {
    features.push({ kind: "team", label: "Unlimited team members" });
  }
  features.push(
    { kind: "cpu", label: `${limits.cpuCoresMaxPerInstance} vCPU per instance` },
    {
      kind: "memory",
      label: `${limits.memoryMibMaxPerInstance / MIB_PER_GIB} GiB memory per instance`,
    },
  );
  if (limits.customDomainsMax >= CUSTOM_DOMAINS_UNLIMITED) {
    features.push({ kind: "domains", label: "Unlimited custom domains" });
  } else if (limits.customDomainsMax > 0) {
    const max = limits.customDomainsMax;
    features.push({ kind: "domains", label: `${max} custom domain${max === 1 ? "" : "s"}` });
  }
  features.push(
    {
      kind: "autoscale",
      label: `Auto scale up to ${limits.autoscalingReplicasMax} instances per region`,
    },
    { kind: "logs", label: `${limits.logsRetentionDaysMax}-day log retention` },
  );
  return features;
}

export type PlanFeatureRow = PlanFeature & { included: boolean };

const ROW_ORDER: PlanFeatureKind[] = ["team", "cpu", "memory", "domains", "autoscale", "logs"];

export function planFeatures(plan: DeployPlan): PlanFeatureRow[] {
  const limits = limitsByPlan[plan];
  const included = limitFeatures(plan).map((feature) => ({ ...feature, included: true }));
  const missing: PlanFeatureRow[] = [];
  if (!limits.teamEnabled) {
    missing.push({ kind: "team", label: "No team members", included: false });
  }
  if (limits.customDomainsMax === 0) {
    missing.push({ kind: "domains", label: "No custom domains", included: false });
  }
  return [...missing, ...included].sort(
    (a, b) => ROW_ORDER.indexOf(a.kind) - ROW_ORDER.indexOf(b.kind),
  );
}
