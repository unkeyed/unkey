import type { DeploymentStatus } from "@/lib/collections/deploy/deployment-status";
import type {
  Environment as CollectionEnvironment,
  EnvironmentKind,
} from "@/lib/collections/deploy/environments";
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

const MAX_NAME_LENGTH = 63;
const RESERVED_PREFIX = "unkey";

export const connectionNameSchema = z
  .string()
  .trim()
  .min(1, "Name is required")
  .max(MAX_NAME_LENGTH)
  .regex(
    /^[a-z](?:[a-z0-9-]{0,61}[a-z0-9])?$/,
    "Start with a letter and use lowercase letters, numbers, and hyphens",
  )
  .refine((value) => value !== "self", "The name self is reserved")
  .refine((value) => !value.startsWith(RESERVED_PREFIX), "Names starting with unkey are reserved");

export function defaultConnectionName(
  slug: string,
  isTaken: (name: string) => boolean,
): string | undefined {
  const base = baseName(slug);
  for (let attempt = 1; attempt <= 20; attempt++) {
    const suffix = attempt === 1 ? "" : `-${attempt}`;
    const candidate = `${base.slice(0, MAX_NAME_LENGTH - suffix.length).replace(/-+$/, "")}${suffix}`;
    if (connectionNameSchema.safeParse(candidate).success && !isTaken(candidate)) {
      return candidate;
    }
  }
  return undefined;
}

function baseName(slug: string): string {
  const cleaned = slug
    .toLowerCase()
    .replace(/[^a-z0-9-]+/g, "-")
    .replace(/-{2,}/g, "-")
    .replace(/^-+|-+$/g, "");
  const prefixed =
    /^[a-z]/.test(cleaned) && !cleaned.startsWith(RESERVED_PREFIX) ? cleaned : `app-${cleaned}`;
  return prefixed.slice(0, MAX_NAME_LENGTH).replace(/-+$/, "");
}

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
export type TargetColumns = Pick<
  Connection,
  "targetType" | "targetEnvironmentId" | "targetDeploymentId"
>;

export const AUTOMATIC_TARGET_LABEL: Record<EnvironmentKind, string> = {
  production: "Production",
  preview: "Preview, same branch",
};

export function isProduction(environment: Pick<Environment, "kind">): boolean {
  return environment.kind === "production";
}

export function ruleOf(connection: TargetColumns): TargetRule | undefined {
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
  connection: TargetColumns;
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

export function columnsOf(rule: TargetRule): TargetColumns {
  return {
    targetType: rule.targetType,
    targetEnvironmentId: rule.targetType === "environment" ? rule.targetEnvironmentId : null,
    targetDeploymentId: rule.targetType === "deployment" ? rule.targetDeploymentId : null,
  };
}

export function targetLabel({
  connection,
  callerEnvironment,
  environments,
}: {
  connection: TargetColumns;
  callerEnvironment: Pick<Environment, "kind">;
  environments: Environment[];
}): string {
  switch (connection.targetType) {
    case "automatic":
      return AUTOMATIC_TARGET_LABEL[callerEnvironment.kind];
    case "environment":
      return (
        environments.find((env) => env.id === connection.targetEnvironmentId)?.slug ??
        "Deleted environment"
      );
    case "deployment":
      return connection.targetDeploymentId ?? "Unavailable deployment";
  }
}
