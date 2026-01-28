import type { DeploymentNode } from "./types";

export const DefaultNode = ({ node }: { node: DeploymentNode }) => (
  <div className="w-[500px] h-[70px] border border-grayA-4 rounded-[14px] bg-gray-1 flex items-center justify-center">
    {node.label}
  </div>
);
