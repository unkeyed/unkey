export type PaywallReason =
  | "team"
  | "deploy"
  | "custom-domains"
  | "api-limit"
  | "compute-plan"
  | "api-plan"
  | "choose-plan";

export type PaywallProduct = "compute" | "api";

export type PaywallCopy = {
  title: string;
  description: string;
  products: PaywallProduct[];
  manage: boolean;
};

export function paywallCopy(reason: PaywallReason): PaywallCopy {
  switch (reason) {
    case "team":
      return {
        title: "Invite your team",
        description: "Team members come with Pro, Business or a paid API plan.",
        products: ["compute", "api"],
        manage: false,
      };
    case "deploy":
      return {
        title: "Choose a Compute plan",
        description: "Upgrade your plan to deploy on Unkey.",
        products: ["compute"],
        manage: false,
      };
    case "custom-domains":
      return {
        title: "Add more custom domains",
        description: "Upgrade your plan to add more custom domains.",
        products: ["compute"],
        manage: false,
      };
    case "api-limit":
      return {
        title: "Raise your API limit",
        description: "Upgrade your plan to raise your monthly API limit.",
        products: ["api"],
        manage: false,
      };
    case "compute-plan":
      return {
        title: "Compute plans",
        description: "Pick the Compute plan that fits your workloads.",
        products: ["compute"],
        manage: true,
      };
    case "api-plan":
      return {
        title: "API plans",
        description: "Pick a plan for your monthly key verifications and ratelimits.",
        products: ["api"],
        manage: true,
      };
    case "choose-plan":
      return {
        title: "Choose your plan",
        description: "Pick a plan to get started, or stay on the free tier for now.",
        products: ["compute", "api"],
        manage: false,
      };
  }
}

export function availableProducts(
  products: PaywallProduct[],
  { computeEnabled }: { computeEnabled: boolean },
): PaywallProduct[] {
  return products.filter((product) => product !== "compute" || computeEnabled);
}

export type ProductUsage = { compute: number; api: number };

export function defaultProduct(
  products: PaywallProduct[],
  usage: ProductUsage,
): PaywallProduct | undefined {
  const preferred: PaywallProduct =
    usage.compute > 0 ? "compute" : usage.api > 0 ? "api" : "compute";
  return products.includes(preferred) ? preferred : products[0];
}
