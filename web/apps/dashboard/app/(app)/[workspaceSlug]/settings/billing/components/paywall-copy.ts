import type { DeployPlan } from "@/lib/stripe/deployPlan";
import type { PlanFeatureKind } from "./plan-features";

export type PaywallReason = "team" | "deploy" | "custom-domains" | "api-limit";

export type PaywallProduct = "compute" | "api";

export type PaywallCopy = {
  title: string;
  description: string;
  products: PaywallProduct[];
  recommendedPlan?: DeployPlan;
  highlight?: PlanFeatureKind;
};

export function paywallCopy(reason: PaywallReason): PaywallCopy {
  switch (reason) {
    case "team":
      return {
        title: "Invite your team",
        description: "Upgrade your plan to add team members.",
        products: ["compute", "api"],
        recommendedPlan: "pro",
        highlight: "team",
      };
    case "deploy":
      return {
        title: "Choose a Compute plan",
        description: "Upgrade your plan to deploy on Unkey.",
        products: ["compute"],
        recommendedPlan: "pro",
      };
    case "custom-domains":
      return {
        title: "Add more custom domains",
        description: "Upgrade your plan to add more custom domains.",
        products: ["compute"],
        recommendedPlan: "pro",
        highlight: "domains",
      };
    case "api-limit":
      return {
        title: "Raise your API limit",
        description: "Upgrade your plan to raise your monthly API limit.",
        products: ["api"],
      };
  }
}
