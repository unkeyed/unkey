import type { DeploymentStatus } from "@/lib/collections/deploy/deployment-status";
import {
  type Environment as CollectionEnvironment,
  ENVIRONMENT_KIND,
} from "@/lib/collections/deploy/environments";
import type { AppConnection } from "@unkey/api/models/components";
import { match } from "@unkey/match";
import { z } from "zod";

export const targetRuleSchema = z.discriminatedUnion("targetType", [
  z.object({ targetType: z.literal("automatic") }),
  z.object({ targetType: z.literal("environment"), targetEnvironmentId: z.string() }),
  z.object({ targetType: z.literal("deployment"), targetDeploymentId: z.string() }),
]);

export type TargetRule = z.infer<typeof targetRuleSchema>;

export type ListedConnection = AppConnection;
type TargetColumns = Pick<
  ListedConnection,
  "targetType" | "targetEnvironmentId" | "targetDeploymentId"
>;

export type Environment = Pick<CollectionEnvironment, "id" | "appId" | "slug" | "kind">;
export type Deployment = {
  id: string;
  appId: string;
  environmentId: string;
  status: DeploymentStatus;
};
export type TargetDeployment = Deployment & { gitBranch: string | null; image: string | null };

export type ConnectionTargets = {
  apps: { id: string; name: string; slug: string }[];
  environments: Environment[];
  deployments: TargetDeployment[];
};

export type TargetContext<D extends Deployment = Deployment> = {
  targetName: string;
  environments: Environment[];
  deployments: D[];
};

type TargetDescription = { text: string; unavailable: boolean };

export const AUTOMATIC_TARGET_LABEL = "Production";

export function isProduction(environment: Pick<Environment, "kind">): boolean {
  return environment.kind === ENVIRONMENT_KIND.production;
}

function available(text: string): TargetDescription {
  return { text, unavailable: false };
}

function unavailable(text: string): TargetDescription {
  return { text, unavailable: true };
}

function followsProduction(targetName: string): TargetDescription {
  return available(`Follows ${targetName}’s live production deployment, including rollbacks.`);
}

export function ruleOf(columns: TargetColumns): TargetRule {
  return match(columns.targetType)
    .with("automatic", (): TargetRule => ({ targetType: "automatic" }))
    .with(
      "environment",
      (): TargetRule => ({
        targetType: "environment",
        targetEnvironmentId: columns.targetEnvironmentId ?? "",
      }),
    )
    .with(
      "deployment",
      (): TargetRule => ({
        targetType: "deployment",
        targetDeploymentId: columns.targetDeploymentId ?? "",
      }),
    )
    .exhaustive();
}

export function sameRule(a: TargetRule, b: TargetRule): boolean {
  return match(a)
    .with({ targetType: "automatic" }, () => b.targetType === "automatic")
    .with(
      { targetType: "environment" },
      (r) => b.targetType === "environment" && b.targetEnvironmentId === r.targetEnvironmentId,
    )
    .with(
      { targetType: "deployment" },
      (r) => b.targetType === "deployment" && b.targetDeploymentId === r.targetDeploymentId,
    )
    .exhaustive();
}

export function targetError(rule: TargetRule): string | null {
  return match(rule)
    .with({ targetType: "automatic" }, () => null)
    .with({ targetType: "environment" }, (r) =>
      r.targetEnvironmentId === "" ? "Choose an environment" : null,
    )
    .with({ targetType: "deployment" }, (r) =>
      r.targetDeploymentId === "" ? "Choose a deployment" : null,
    )
    .exhaustive();
}

export function describeTarget(rule: TargetRule, context: TargetContext): TargetDescription {
  const { targetName } = context;
  return match(rule)
    .with({ targetType: "automatic" }, () => followsProduction(targetName))
    .with({ targetType: "environment" }, ({ targetEnvironmentId }) => {
      const environment = context.environments.find((env) => env.id === targetEnvironmentId);
      if (!environment) {
        return unavailable(
          `The selected ${targetName} environment was deleted. Choose another target.`,
        );
      }
      return isProduction(environment)
        ? followsProduction(targetName)
        : available(`Connects to ${targetName}’s latest ${environment.slug} version.`);
    })
    .with({ targetType: "deployment" }, ({ targetDeploymentId }) => {
      const deployment = context.deployments.find((item) => item.id === targetDeploymentId);
      if (!deployment) {
        return unavailable(
          `The pinned ${targetName} deployment is unavailable. Choose another target.`,
        );
      }
      if (deployment.status !== "ready") {
        return unavailable(
          `The pinned ${targetName} deployment is ${deployment.status === "stopped" ? "stopped" : "unavailable"}. Choose another target.`,
        );
      }
      return available(`Always connects to ${targetName} deployment ${deployment.id}.`);
    })
    .exhaustive();
}

export function targetLabel(rule: TargetRule, context: TargetContext): string {
  return match(rule)
    .with({ targetType: "automatic" }, () => AUTOMATIC_TARGET_LABEL)
    .with(
      { targetType: "environment" },
      ({ targetEnvironmentId }) =>
        context.environments.find((env) => env.id === targetEnvironmentId)?.slug ??
        "Deleted environment",
    )
    .with(
      { targetType: "deployment" },
      ({ targetDeploymentId }) => targetDeploymentId || "Unavailable deployment",
    )
    .exhaustive();
}

export type ConnectionTarget = TargetDescription & { label: string };

export function connectionTarget(
  connection: ListedConnection,
  targets: ConnectionTargets,
): ConnectionTarget {
  const rule = ruleOf(connection);
  const context: TargetContext = {
    targetName: connection.targetAppName,
    environments: targets.environments,
    deployments: targets.deployments,
  };
  return { label: targetLabel(rule, context), ...describeTarget(rule, context) };
}
