"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import type { Environment } from "@/lib/collections/deploy/environments";
import {
  environmentRedirectPath,
  resolveEnvironmentRoute,
} from "@/lib/navigation/environment-route";
import type { AppScope } from "@/lib/navigation/routes/projects";
import {
  RedirectType,
  notFound,
  redirect,
  useParams,
  usePathname,
  useSearchParams,
} from "next/navigation";
import { type PropsWithChildren, createContext, useContext } from "react";
import { useAppId, useProjectData } from "../data-provider";

type AppEnvironmentContextValue = {
  environment: Environment;
  environments: Environment[];
  slugFor: (environmentId: string) => string;
};

const AppEnvironmentContext = createContext<AppEnvironmentContextValue | null>(null);

export function AppEnvironmentProvider({ children }: PropsWithChildren) {
  const params = useParams<{ appId: string; environmentSlug: string }>();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const { environments, isEnvironmentsLoading } = useProjectData();

  if (isEnvironmentsLoading) {
    return null;
  }

  const route = resolveEnvironmentRoute(params.environmentSlug, environments);
  if (route.kind === "notFound") {
    notFound();
  }
  if (route.kind === "legacy" || route.kind === "unknown") {
    const search = searchParams.toString();
    const target = environmentRedirectPath(pathname, params.appId, route);
    redirect(search ? `${target}?${search}` : target, RedirectType.replace);
  }

  const { environment } = route;
  const value: AppEnvironmentContextValue = {
    environment,
    environments,
    slugFor: (environmentId) =>
      environments.find((candidate) => candidate.id === environmentId)?.slug ?? environment.slug,
  };
  return <AppEnvironmentContext.Provider value={value}>{children}</AppEnvironmentContext.Provider>;
}

export function useAppEnvironment(): AppEnvironmentContextValue {
  const context = useContext(AppEnvironmentContext);
  if (!context) {
    throw new Error(
      "useAppEnvironment must be used inside an /apps/[appId]/[environmentSlug] route",
    );
  }
  return context;
}

/** Every id a route builder under /apps/[appId]/[environmentSlug] needs. */
export function useAppScope(): AppScope {
  const workspace = useWorkspaceNavigation();
  const { projectId } = useProjectData();
  const appId = useAppId();
  const { environment } = useAppEnvironment();
  return { workspaceSlug: workspace.slug, projectId, appId, environmentSlug: environment.slug };
}
