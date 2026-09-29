export type TargetRule =
  | { targetType: "automatic" }
  | { targetType: "environment"; targetEnvironmentId: string }
  | { targetType: "deployment"; targetDeploymentId: string };

export type Binding = {
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

export type BindingTargets = {
  apps: { id: string; name: string; slug: string }[];
  environments: Environment[];
  deployments: (Deployment & { gitBranch: string | null; image: string | null })[];
};
export type ListedBinding = Binding & { name: string; targetAppName: string };

export function isProduction(environment: Pick<Environment, "slug" | "kind">): boolean {
  return environment.kind === "production" && environment.slug === "production";
}

export function ruleOf(binding: Binding): TargetRule | undefined {
  switch (binding.targetType) {
    case "automatic":
      return { targetType: "automatic" };
    case "environment":
      return binding.targetEnvironmentId
        ? { targetType: "environment", targetEnvironmentId: binding.targetEnvironmentId }
        : undefined;
    case "deployment":
      return binding.targetDeploymentId
        ? { targetType: "deployment", targetDeploymentId: binding.targetDeploymentId }
        : undefined;
  }
}

export function describeTarget({
  binding,
  targetName,
  callerEnvironment,
  environments,
  deployments,
}: {
  binding: Binding;
  targetName: string;
  callerEnvironment: Pick<Environment, "slug" | "kind">;
  environments: Environment[];
  deployments: Deployment[];
}): { text: string; unavailable: boolean } {
  if (binding.targetType === "automatic") {
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
  if (binding.targetType === "environment") {
    const environment = environments.find((env) => env.id === binding.targetEnvironmentId);
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
  const deployment = deployments.find((item) => item.id === binding.targetDeploymentId);
  if (!deployment || deployment.status !== "ready") {
    return {
      text: `The pinned ${targetName} deployment is stopped. Choose another target.`,
      unavailable: true,
    };
  }
  return {
    text: `Always connects to ${targetName} deployment ${deployment.id}.`,
    unavailable: false,
  };
}
