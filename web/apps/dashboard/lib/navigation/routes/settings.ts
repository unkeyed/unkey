/**
 * Route builders for the /settings area, exposed as one nested object so call
 * sites read like the url hierarchy: `routes.settings.stripe.portal(scope)`.
 * Every navigable result goes through buildRoute, which checks the bracket
 * pattern against Next's generated route table (typedRoutes) and types the
 * params from the generated ParamMap.
 */
import type { Route } from "next";
import { type WorkspaceScope, buildRoute } from "./shared";

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

  stripe: {
    portal({ workspaceSlug }: WorkspaceScope): Route {
      return buildRoute("/[workspaceSlug]/settings/billing/stripe/portal", { workspaceSlug });
    },

    checkout({ workspaceSlug }: WorkspaceScope): Route {
      return buildRoute("/[workspaceSlug]/settings/billing/stripe/checkout", { workspaceSlug });
    },
  },
};
