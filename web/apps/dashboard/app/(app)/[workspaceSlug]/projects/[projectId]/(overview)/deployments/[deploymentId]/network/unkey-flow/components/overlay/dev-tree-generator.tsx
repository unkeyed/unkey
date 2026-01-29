import type { DeploymentNode } from "../nodes/types";
import { InternalDevTreeGenerator } from "../simulate/tree-generate";

type DevTreeGeneratorProps = {
  deploymentId: string;
  onTreeGenerate: (tree: DeploymentNode) => void;
  defaultTree: DeploymentNode;
};

export const DevTreeGenerator = ({
  deploymentId,
  onTreeGenerate,
  defaultTree,
}: DevTreeGeneratorProps) => {
  return process.env.NODE_ENV === "development" ? (
    <InternalDevTreeGenerator
      deploymentId={deploymentId}
      onGenerate={onTreeGenerate}
      onReset={() => onTreeGenerate(defaultTree)}
    />
  ) : null;
};
