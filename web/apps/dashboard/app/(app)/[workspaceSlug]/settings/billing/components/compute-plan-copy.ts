import type { DeployPlan } from "@/lib/stripe/deployPlan";
import { COMPUTE_BILLING_DOCS } from "@/lib/support";

/** Marketing copy for the Compute plan picker. */
export const PLAN_BLURBS: Record<DeployPlan, string> = {
  starter: "For hobby projects and testing ideas",
  pro: "For growing apps in production",
  business: "For teams scaling with confidence",
};

export const CREDITS_INFO = "Every plan includes monthly usage credit.";
export const CREDITS_LINK_LABEL = "See how credits work";
export const CREDITS_LINK_HREF = `${COMPUTE_BILLING_DOCS}#how-the-bill-works`;
