"use client";
import { collection } from "@/lib/collections";
import { type InitialQueryBuilder, and, eq } from "@tanstack/react-db";

// Legacy pages only. Everything else reads through @/lib/deploy-data.
export const appDeploymentQueryFor =
  (projectId: string, appId: string, deploymentId: string) => (q: InitialQueryBuilder) =>
    q
      .from({ deployment: collection.deployments })
      .where(({ deployment }) =>
        and(
          eq(deployment.projectId, projectId),
          eq(deployment.appId, appId),
          eq(deployment.id, deploymentId),
        ),
      );
