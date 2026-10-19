import { routes } from "@/lib/navigation/routes";
import type { WorkspaceScope } from "@/lib/navigation/routes/shared";
import { IconCubeOutline18, IconNodesOutline18, type IconProps } from "@unkey/icons";
import type { Route } from "next";
import type { ComponentType } from "react";

export const INTENT_ORDER = ["compute", "api"] as const;

export type Intent = (typeof INTENT_ORDER)[number];

type IntentOption = {
  title: string;
  description: string;
  icon: ComponentType<IconProps>;
  landing: (scope: WorkspaceScope) => Route;
};

export const INTENTS: Record<Intent, IntentOption> = {
  compute: {
    title: "Compute",
    description: "Deploy and run your apps on Unkey. Push to git and get a live URL.",
    icon: IconCubeOutline18,
    landing: routes.projects.list,
  },
  api: {
    title: "API management",
    description: "Issue and verify API keys, set rate limits, and track usage.",
    icon: IconNodesOutline18,
    landing: routes.apis.list,
  },
};
