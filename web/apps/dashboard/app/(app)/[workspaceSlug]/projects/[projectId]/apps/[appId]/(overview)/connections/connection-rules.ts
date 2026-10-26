import type { DeploymentStatus } from "@/lib/collections/deploy/deployment-status";
import type { Environment as CollectionEnvironment } from "@/lib/collections/deploy/environments";
import type { AppConnection } from "@unkey/api/models/components";
import { z } from "zod";

export type TargetRule =
  | { targetType: "automatic" }
  | { targetType: "environment"; targetEnvironmentId: string }
  | { targetType: "deployment"; targetDeploymentId: string };

export type Connection = {
  id: string;
  appId: string;
  environmentId: string;
  targetAppId: string;
  targetType: TargetRule["targetType"];
  targetEnvironmentId?: string | null;
  targetDeploymentId?: string | null;
};

export type Environment = Pick<CollectionEnvironment, "id" | "appId" | "slug" | "kind">;
export type Deployment = {
  id: string;
  appId: string;
  environmentId: string;
  status: DeploymentStatus;
};

export type ConnectionTargets = {
  apps: { id: string; name: string; slug: string }[];
  environments: Environment[];
  deployments: (Deployment & { gitBranch: string | null; image: string | null })[];
};
export type ListedConnection = AppConnection;

export const connectionNameSchema = z
  .string()
  .trim()
  .min(1, "Name is required")
  .max(63)
  .regex(
    /^[a-z](?:[a-z0-9-]{0,61}[a-z0-9])?$/,
    "Start with a letter and use lowercase letters, numbers, and hyphens",
  )
  .refine((value) => value !== "self", "The name self is reserved")
  .refine((value) => !value.startsWith("unkey"), "Names starting with unkey are reserved");

export function connectionHostVariable(name: string): string {
  return `${name.replaceAll("-", "_").toUpperCase()}_HOST`;
}

export function mergeDeploymentTargets<T extends { id: string }>(pins: T[], choices: T[]): T[] {
  const seen = new Set<string>();
  return [...pins, ...choices].filter((deployment) => {
    if (seen.has(deployment.id)) {
      return false;
    }
    seen.add(deployment.id);
    return true;
  });
}

export function isProduction(environment: Pick<Environment, "kind">): boolean {
  return environment.kind === "production";
}

export function ruleOf(connection: Connection): TargetRule | undefined {
  switch (connection.targetType) {
    case "automatic":
      return { targetType: "automatic" };
    case "environment":
      return connection.targetEnvironmentId
        ? { targetType: "environment", targetEnvironmentId: connection.targetEnvironmentId }
        : undefined;
    case "deployment":
      return connection.targetDeploymentId
        ? { targetType: "deployment", targetDeploymentId: connection.targetDeploymentId }
        : undefined;
  }
}

export function describeTarget({
  connection,
  targetName,
  callerEnvironment,
  environments,
  deployments,
}: {
  connection: Connection;
  targetName: string;
  callerEnvironment: Pick<Environment, "kind">;
  environments: Environment[];
  deployments: Deployment[];
}): { text: string; unavailable: boolean } {
  if (connection.targetType === "automatic") {
    if (isProduction(callerEnvironment)) {
      return {
        text: `Follows ${targetName}’s live production deployment, including rollbacks.`,
        unavailable: false,
      };
    }
    return {
      text: `Connects to ${targetName}’s preview from the same Git branch, if there is one.`,
      unavailable: false,
    };
  }
  if (connection.targetType === "environment") {
    const environment = environments.find((env) => env.id === connection.targetEnvironmentId);
    if (!environment) {
      return {
        text: `The selected ${targetName} environment was deleted. Choose another target.`,
        unavailable: true,
      };
    }
    return isProduction(environment)
      ? {
          text: `Follows ${targetName}’s live production deployment, including rollbacks.`,
          unavailable: false,
        }
      : {
          text: `Connects to ${targetName}’s latest ${environment.slug} version.`,
          unavailable: false,
        };
  }
  const deployment = deployments.find((item) => item.id === connection.targetDeploymentId);
  if (!deployment) {
    return {
      text: `The pinned ${targetName} deployment is unavailable. Choose another target.`,
      unavailable: true,
    };
  }
  if (deployment.status !== "ready") {
    return {
      text: `The pinned ${targetName} deployment is ${deployment.status === "stopped" ? "stopped" : "unavailable"}. Choose another target.`,
      unavailable: true,
    };
  }
  return {
    text: `Always connects to ${targetName} deployment ${deployment.id}.`,
    unavailable: false,
  };
}
