import type { DeployPlan } from "@/lib/stripe/deployPlan";

export type PaywallReason = "team" | "deploy" | "custom-domains" | "api-limit";

export type PaywallProduct = "compute" | "api";

export type PaywallCopy = {
  title: string;
  description: string;
  products: PaywallProduct[];
  recommendedPlan?: DeployPlan;
};

export function paywallCopy(reason: PaywallReason, currentPlan: DeployPlan | null): PaywallCopy {
  switch (reason) {
    case "team":
      return {
        title: "Invite your team",
        description:
          currentPlan === "starter"
            ? "Starter doesn't include team members. Upgrade to Pro or Business, or add any API plan."
            : "Team members come with the Pro and Business Compute plans, and with every API plan.",
        products: ["compute", "api"],
        recommendedPlan: "pro",
      };
    case "deploy":
      return {
        title: "Choose a Compute plan",
        description:
          "Deploying on Unkey requires a Compute plan. Every plan includes usage credits equal to its fee.",
        products: ["compute"],
        recommendedPlan: "pro",
      };
    case "custom-domains":
      return {
        title: "Add more custom domains",
        description:
          currentPlan === "starter"
            ? "Starter includes 1 custom domain. Pro and Business include unlimited custom domains."
            : "Custom domains start on Starter. Pro and Business include unlimited custom domains.",
        products: ["compute"],
        recommendedPlan: "pro",
      };
    case "api-limit":
      return {
        title: "Raise your API limit",
        description:
          "Pick a plan with more monthly key verifications and ratelimits. Every API plan includes team members.",
        products: ["api"],
      };
  }
}
