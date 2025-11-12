"use client";

import { TreeConnectionLine, TreeLayout } from "./components/unkey-flow";
import { InfiniteCanvas } from "./components/unkey-flow/components/canvas/infinite-canvas";
import type { DeploymentNode } from "./components/unkey-flow/components/nodes/types";
import { deploymentTree } from "./components/unkey-flow/components/nodes/mock-data";
import {
  InstanceNode,
  RegionNode,
} from "./components/unkey-flow/components/nodes/deploy-node";

const OriginNode = ({ node }: { node: DeploymentNode }) => (
  <div className="w-[70px] h-[20px] ring-4 rounded-full ring-grayA-5 bg-gray-9 flex items-center justify-center p-2.5 shadow-sm">
    <div className="font-mono text-[9px] font-medium text-white leading-[6px]">
      {node.label}
    </div>
  </div>
);

const DefaultNode = ({ node }: { node: DeploymentNode }) => (
  <div className="w-[500px] h-[70px] border border-grayA-4 rounded-[14px] bg-gray-1 flex items-center justify-center">
    {node.label}
  </div>
);

export default function DeploymentDetailsPage() {
  return (
    <InfiniteCanvas>
      <TreeLayout
        data={deploymentTree}
        nodeSpacing={{ x: 25, y: 150 }}
        renderNode={(node, _, parent) => {
          switch (node.metadata.type) {
            case "origin":
              return <OriginNode node={node} />;
            case "region":
              return (
                <RegionNode
                  node={
                    node as DeploymentNode & { metadata: { type: "region" } }
                  }
                />
              );
            case "instance":
              if (!parent?.id) {
                throw new Error("Instance node requires parent region");
              }
              return (
                <InstanceNode
                  node={
                    node as DeploymentNode & { metadata: { type: "instance" } }
                  }
                  parentRegionId={parent.id}
                />
              );
            default:
              return <DefaultNode node={node} />;
          }
        }}
        renderConnection={(from, to, parent, child) => {
          const childIndex =
            parent.children?.findIndex((c) => c.id === child.id) ?? 0;
          const childCount = parent.children?.length ?? 1;
          const xOffset = (childIndex - (childCount - 1) / 2) * 5;

          return (
            <TreeConnectionLine
              key={`${parent.id}-${child.id}`}
              from={{ x: from.x + xOffset, y: from.y }}
              to={{ x: to.x, y: to.y }}
            />
          );
        }}
      />
    </InfiniteCanvas>
  );
}
