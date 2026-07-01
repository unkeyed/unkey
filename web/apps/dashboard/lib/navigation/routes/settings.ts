/**
 * Route builders for the /settings area, exposed as one nested object so call
 * sites read like the url hierarchy: `routes.settings.stripe.portal(scope)`.
 * Every navigable result goes through buildRoute, which checks the bracket
 * pattern against Next's generated route table (typedRoutes) and types the
 * params from the generated ParamMap.
 */
import type { Route } from "next";
import { type WorkspaceScope, buildRoute } from "./shared";

export type CheckoutIntent = "compute" | "api" | "payment" | "deploy";

/** Compute-plan tiers carried through the deploy-gate checkout round-trip. */
export type DeployCheckoutPlan = "starter" | "pro" | "business";
/** Where the deploy-gate dialog was opened from, for post-subscribe routing. */
export type DeployCheckoutOrigin = "create" | "banner" | "billing";

export const settingsRoutes = {
  general({ workspaceSlug }: WorkspaceScope): Route {
    return buildRoute("/[workspaceSlug]/settings/general", { workspaceSlug });
  },

  team({ workspaceSlug }: WorkspaceScope): Route {
    return buildRoute("/[workspaceSlug]/settings/team", { workspaceSlug });
  },

  rootKeys({ workspaceSlug }: WorkspaceScope): Route {
    return buildRoute("/[workspaceSlug]/settings/root-keys", { workspaceSlug });
  },

  billing({ workspaceSlug }: WorkspaceScope): Route {
    return buildRoute("/[workspaceSlug]/settings/billing", { workspaceSlug });
  },

  security({ workspaceSlug }: WorkspaceScope): Route {
    return buildRoute("/[workspaceSlug]/settings/security", { workspaceSlug });
  },

  stripe: {
    portal({ workspaceSlug }: WorkspaceScope): Route {
      return buildRoute("/[workspaceSlug]/stripe/portal", { workspaceSlug });
    },

    checkout({
      workspaceSlug,
      intent,
      plan,
      from,
    }: WorkspaceScope & {
      intent?: CheckoutIntent;
      plan?: DeployCheckoutPlan;
      from?: DeployCheckoutOrigin;
    }): Route {
      const query =
        intent || plan || from
          ? { ...(intent ? { intent } : {}), ...(plan ? { plan } : {}), ...(from ? { from } : {}) }
          : undefined;
      return buildRoute("/[workspaceSlug]/stripe/checkout", { workspaceSlug }, query);
    },
  },
};
