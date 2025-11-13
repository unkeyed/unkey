import type { DeploymentNode } from "./types";

export const OriginNode = ({ node }: { node: DeploymentNode }) => (
  <div className="w-[70px] h-[20px] ring-4 rounded-full ring-grayA-5 bg-gray-9 flex items-center justify-center p-2.5 shadow-sm">
    <div className="font-mono text-[9px] font-medium text-white leading-[6px]">
      {node.label}
    </div>
  </div>
);
