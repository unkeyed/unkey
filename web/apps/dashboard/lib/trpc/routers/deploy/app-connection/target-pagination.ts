export type TargetDeployment = {
  id: string;
  appId: string;
  createdAt: number;
};

export function pageTargetDeployments<T extends TargetDeployment>(
  deployments: T[],
  targetAppId: string,
  limit: number,
) {
  const ordered = deployments
    .filter((deployment) => deployment.appId === targetAppId)
    .toSorted(
      (left, right) =>
        right.createdAt - left.createdAt || right.id.localeCompare(left.id),
    );
  const page = ordered.slice(0, limit);
  const last = page.at(-1);
  return {
    page,
    nextCursor:
      ordered.length > limit && last
        ? { createdAt: last.createdAt, id: last.id }
        : null,
  };
}

export function mergeDeploymentTargets<T extends Pick<TargetDeployment, "id">>(
  pins: T[],
  choices: T[],
): T[] {
  return [...pins, ...choices].filter(
    (deployment, index, deployments) =>
      deployments.findIndex(({ id }) => id === deployment.id) === index,
  );
}
