import { CUSTOM_DOMAINS_UNLIMITED, limitsByPlan } from "@/lib/limits";
import { DEPLOY_PLANS, type DeployPlan } from "@/lib/stripe/deployPlan";

export type PlanFeatureKind =
  | "git"
  | "preview"
  | "rollback"
  | "team"
  | "cpu"
  | "memory"
  | "domains"
  | "autoscale"
  | "logs";

export type PlanFeature = {
  kind: PlanFeatureKind;
  label: string;
};

export type PlanFeatureSet = {
  inheritsFrom: DeployPlan | null;
  features: PlanFeature[];
};

const BASE_FEATURES: PlanFeature[] = [
  { kind: "git", label: "Git push to deploy" },
  { kind: "preview", label: "Preview deploy per PR" },
  { kind: "rollback", label: "Instant rollback" },
];

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
    { kind: "autoscale", label: `Up to ${limits.autoscalingReplicasMax} instances per region` },
    { kind: "logs", label: `${limits.logsRetentionDaysMax}-day log retention` },
  );
  return features;
}

export function computePlanFeatures(plan: DeployPlan): PlanFeatureSet {
  const previous = DEPLOY_PLANS[DEPLOY_PLANS.indexOf(plan) - 1];
  if (previous === undefined) {
    return { inheritsFrom: null, features: [...BASE_FEATURES, ...limitFeatures(plan)] };
  }
  const inherited = new Set(limitFeatures(previous).map((feature) => feature.label));
  return {
    inheritsFrom: previous,
    features: limitFeatures(plan).filter((feature) => !inherited.has(feature.label)),
  };
}

export type PlanFeatureRow = PlanFeature & { included: boolean };

export function fullPlanFeatures(plan: DeployPlan): PlanFeatureRow[] {
  const limits = limitsByPlan[plan];
  const included = limitFeatures(plan).map((feature) => ({ ...feature, included: true }));
  const missing: PlanFeatureRow[] = [];
  if (!limits.teamEnabled) {
    missing.push({ kind: "team", label: "No team members", included: false });
  }
  if (limits.customDomainsMax === 0) {
    missing.push({ kind: "domains", label: "No custom domains", included: false });
  }
  return [...missing, ...included];
}
