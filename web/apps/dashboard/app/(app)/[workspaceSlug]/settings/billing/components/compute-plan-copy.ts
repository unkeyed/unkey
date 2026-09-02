import type { IconProps } from "nucleo-ui-outline-18";
import {
  IconArrowDottedRotateAnticlockwiseOutline18,
  IconChartActivityOutline18,
  IconCodeBranchOutline18,
  IconEyeOutline18,
} from "nucleo-ui-outline-18";
import type { ComponentType } from "react";
import type { DeployPlan } from "@/lib/stripe/deployPlan";
import { BILLING_DOCS } from "@/lib/support";

/** Marketing copy for the Compute plan picker. */
export const PLAN_BLURBS: Record<DeployPlan, string> = {
  starter: "For hobby projects and testing ideas",
  pro: "For growing apps in production",
  business: "For teams scaling with confidence",
};

export const ALL_PLANS_INCLUDE = [
  "Git push to deploy",
  "Preview deploy per PR",
  "Instant rollback",
  "Auto-scaling",
] as const;

type ComputeFeature = {
  Icon: ComponentType<IconProps>;
  title: string;
  description: string;
};

export const FEATURES: ComputeFeature[] = [
  {
    Icon: IconCodeBranchOutline18,
    title: "Git push to deploy",
    description: "Every commit you push deploys automatically.",
  },
  {
    Icon: IconEyeOutline18,
    title: "Preview deploy per PR",
    description: "Every pull request gets its own isolated preview URL.",
  },
  {
    Icon: IconArrowDottedRotateAnticlockwiseOutline18,
    title: "Instant rollback",
    description: "Roll back to any previous deploy in one click.",
  },
  {
    Icon: IconChartActivityOutline18,
    title: "Auto-scaling",
    description: "Automatic scaling that follows demand.",
  },
];

export const CREDITS_INFO = "Every plan includes monthly usage credit.";
export const CREDITS_LINK_LABEL = "See how credits work";
export const CREDITS_LINK_HREF = BILLING_DOCS;
export const COMPUTE_PLANS_LINK_HREF = BILLING_DOCS;
