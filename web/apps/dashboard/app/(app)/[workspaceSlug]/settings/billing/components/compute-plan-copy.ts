import type { DeployPlan } from "@/lib/stripe/deployPlan";
import { ArrowDottedRotateAnticlockwise, ChartActivity, CodeBranch, Eye } from "@unkey/icons";
import type { IconProps } from "@unkey/icons";

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
  Icon: (props: IconProps) => React.JSX.Element;
  title: string;
  description: string;
};

export const FEATURES: ComputeFeature[] = [
  {
    Icon: CodeBranch,
    title: "Git push to deploy",
    description: "Every commit you push deploys automatically.",
  },
  {
    Icon: Eye,
    title: "Preview deploy per PR",
    description: "Every pull request gets its own isolated preview URL.",
  },
  {
    Icon: ArrowDottedRotateAnticlockwise,
    title: "Instant rollback",
    description: "Roll back to any previous deploy in one click.",
  },
  {
    Icon: ChartActivity,
    title: "Auto-scaling",
    description: "Scales with traffic automatically, down to zero when idle.",
  },
];

export const CREDITS_INFO = "Every plan includes monthly usage credit.";
export const CREDITS_LINK_LABEL = "See how credits work";
// TODO(dave): real docs URL for "how credits work" is not decided yet.
export const CREDITS_LINK_HREF = "#";
// TODO(dave): real docs URL for "Compute plans" is not decided yet.
export const COMPUTE_PLANS_LINK_HREF = "#";
