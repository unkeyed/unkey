"use client";

import { routes } from "@/lib/navigation/routes";
import { RedirectType, redirect } from "next/navigation";
import { useAppEnvironment, useAppScope } from "../../environment-context";
import { useDeployment } from "./layout-provider";

// A deployment opened under another environment's url moves to its own.
export function DeploymentEnvironmentGuard() {
  const { deployment } = useDeployment();
  const { environment, slugFor } = useAppEnvironment();
  const scope = useAppScope();
  if (deployment.environmentId !== environment.id) {
    redirect(
      routes.projects.apps.deployment({
        ...scope,
        environmentSlug: slugFor(deployment.environmentId),
        deploymentId: deployment.id,
      }),
      RedirectType.replace,
    );
  }
  return null;
}
