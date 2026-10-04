export type TargetDeployment = {
  id: string;
  createdAt: number;
};

export function pageTargetDeployments<T extends TargetDeployment>(deployments: T[], limit: number) {
  const page = deployments.slice(0, limit);
  const last = page.at(-1);
  return {
    page,
    nextCursor:
      deployments.length > limit && last ? { createdAt: last.createdAt, id: last.id } : null,
  };
}

export function mergeDeploymentTargets<T extends Pick<TargetDeployment, "id">>(
  pins: T[],
  choices: T[],
): T[] {
  const seen = new Set<string>();
  return [...pins, ...choices].filter((deployment) => {
    if (seen.has(deployment.id)) {
      return false;
    }
    seen.add(deployment.id);
    return true;
  });
}
