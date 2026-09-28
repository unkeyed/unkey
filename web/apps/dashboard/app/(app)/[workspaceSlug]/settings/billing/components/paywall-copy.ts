import type { DeployPlan } from "@/lib/stripe/deployPlan";

export type PaywallReason = "team" | "deploy" | "custom-domains" | "api-limit";

export type PaywallProduct = "compute" | "api";

export type PaywallCopy = {
  title: string;
  description: string;
  products: PaywallProduct[];
  recommendedPlan?: DeployPlan;
};

export function paywallCopy(reason: PaywallReason): PaywallCopy {
  switch (reason) {
    case "team":
      return {
        title: "Invite your team",
        description: "Upgrade your plan to add team members.",
        products: ["compute", "api"],
        recommendedPlan: "pro",
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
      };
    case "api-limit":
      return {
        title: "Raise your API limit",
        description: "Upgrade your plan to raise your monthly API limit.",
        products: ["api"],
      };
  }
}

export function availableProducts(
  products: PaywallProduct[],
  { computeEnabled }: { computeEnabled: boolean },
): PaywallProduct[] {
  return products.filter((product) => product !== "compute" || computeEnabled);
}
