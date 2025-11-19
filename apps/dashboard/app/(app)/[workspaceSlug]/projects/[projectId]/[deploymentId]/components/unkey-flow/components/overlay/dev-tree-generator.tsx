import type { DeploymentNode } from "../nodes/types";
import { generateDeploymentTree } from "../simulate/simulate";
import { InternalDevTreeGenerator } from "../simulate/tree-generate";

export const DEFAULT_TREE = generateDeploymentTree({
  regions: 3,
  instancesPerRegion: { min: 2, max: 3 },
  regionDirection: "vertical",
  instanceDirection: "horizontal",
  healthDistribution: {
    normal: 80,
    unstable: 10,
    degraded: 5,
    unhealthy: 5,
    recovering: 0,
    health_syncing: 0,
    unknown: 0,
    disabled: 0,
  },
});

export const DevTreeGenerator = ({
  onTreeGenerate,
}: {
  onTreeGenerate: (tree: DeploymentNode) => void;
}) => {
  return process.env.NODE_ENV === "development" ? (
    <InternalDevTreeGenerator
      onGenerate={(config) => onTreeGenerate(generateDeploymentTree(config))}
      onReset={() => onTreeGenerate(DEFAULT_TREE)}
    />
  ) : undefined;
};
