import {
  DEPLOY_CHECKOUT_ORIGINS,
  type DeployCheckoutOrigin,
  routes,
} from "@/lib/navigation/routes";
import { buildRoute } from "@/lib/navigation/routes/shared";
import { withQuery } from "@/lib/navigation/url";
import { DEPLOY_PLANS, type DeployPlan } from "@/lib/stripe/deployPlan";
import type { Route } from "next";

export type UpgradeResult = { kind: "compute"; plan: DeployPlan } | { kind: "api" };

export type CheckoutOutcome = "subscribed" | "none";

const PROJECTS_ORIGINS = new Set<DeployCheckoutOrigin | undefined>(["create", "banner"]);

const SAFE_PATH = /^\/[A-Za-z0-9_\-/]*$/;

export function parseReturnPath(
  raw: string | null | undefined,
  workspaceSlug: string,
): Route | null {
  if (!raw || !SAFE_PATH.test(raw) || raw.includes("//")) {
    return null;
  }
  const root = `/${workspaceSlug}`;
  if (raw !== root && !raw.startsWith(`${root}/`)) {
    return null;
  }
  // Checked above to be a plain path inside this workspace, the only shape Route needs.
  return raw as Route;
}

export function parseUpgradeResult(params: {
  get(name: string): string | null;
}): UpgradeResult | null {
  switch (params.get("upgraded")) {
    case "compute": {
      const rawPlan = params.get("plan");
      const plan = DEPLOY_PLANS.find((known) => known === rawPlan);
      return plan ? { kind: "compute", plan } : null;
    }
    case "api":
      return { kind: "api" };
    default:
      return null;
  }
}

export function upgradeQuery(result: UpgradeResult): Record<string, string> {
  return result.kind === "compute"
    ? { upgraded: result.kind, plan: result.plan }
    : { upgraded: result.kind };
}

export function checkoutReturnPath({
  workspaceSlug,
  outcome,
  intent,
  plan,
  from,
  returnTo,
}: {
  workspaceSlug: string;
  outcome: CheckoutOutcome;
  intent?: string;
  plan?: string;
  from?: string;
  returnTo?: string;
}): Route {
  const billing = routes.settings.billing({ workspaceSlug });
  const returnPath = parseReturnPath(returnTo, workspaceSlug);
  switch (intent) {
    case "deploy": {
      const deployPlan = DEPLOY_PLANS.find((known) => known === plan);
      const origin = DEPLOY_CHECKOUT_ORIGINS.find((known) => known === from);
      if (outcome === "subscribed" && returnPath && deployPlan && !PROJECTS_ORIGINS.has(origin)) {
        return withQuery(returnPath, upgradeQuery({ kind: "compute", plan: deployPlan }));
      }
      return buildRoute(
        "/[workspaceSlug]/projects",
        { workspaceSlug },
        { pendingPlan: deployPlan, from: origin },
      );
    }
    case "api-subscription":
      return outcome === "subscribed"
        ? withQuery(returnPath ?? billing, upgradeQuery({ kind: "api" }))
        : billing;
    default:
      return billing;
  }
}
