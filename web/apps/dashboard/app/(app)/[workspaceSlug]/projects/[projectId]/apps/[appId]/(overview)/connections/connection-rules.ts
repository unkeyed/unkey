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
  targetEnvironmentId: string | null;
  targetDeploymentId: string | null;
};

export type Environment = { id: string; appId: string; slug: string; kind: string };
export type Deployment = { id: string; appId: string; environmentId: string; status: string };

export type ConnectionTargets = {
  apps: { id: string; name: string; slug: string }[];
  environments: Environment[];
  deployments: (Deployment & { gitBranch: string | null; image: string | null })[];
};
export type ListedConnection = Connection & { name: string; targetAppName: string };

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
