"use client";

import { TreeConnectionLine, TreeLayout } from "./components/unkey-flow";
import { InfiniteCanvas } from "./components/unkey-flow/components/canvas/infinite-canvas";
import type { DeploymentNode } from "./components/unkey-flow/components/nodes/types";
import { deploymentTree } from "./components/unkey-flow/components/nodes/mock-data";
import {
  InstanceNode,
  RegionNode,
} from "./components/unkey-flow/components/nodes/deploy-node";
import { OriginNode } from "./components/unkey-flow/components/nodes/origin-node";
import { DefaultNode } from "./components/unkey-flow/components/nodes/default-node";
import { generateDeploymentTree } from "./components/unkey-flow/components/simulate/simulate";
import { useState } from "react";
import { DevTreeGenerator } from "./components/unkey-flow/components/simulate/tree-generate";

export default function DeploymentDetailsPage() {
  const [generatedTree, setGeneratedTree] = useState<DeploymentNode | null>(
    null
  );

  const tree = generatedTree ?? deploymentTree;

  return (
    <InfiniteCanvas
      overlay={
        process.env.NODE_ENV === "development" ? (
          <DevTreeGenerator
            onGenerate={(config) =>
              setGeneratedTree(generateDeploymentTree(config))
            }
            onReset={() => setGeneratedTree(null)}
          />
        ) : undefined
      }
    >
      <TreeLayout
        data={tree}
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
                    node as DeploymentNode & {
                      metadata: { type: "instance" };
                    }
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
