"use client";
import { TreeConnectionLine, TreeLayout } from "./components/unkey-flow";
import { InfiniteCanvas } from "./components/unkey-flow/components/canvas/infinite-canvas";
import type { DeploymentNode } from "./components/unkey-flow/components/nodes/types";
import {
  InstanceNode,
  RegionNode,
} from "./components/unkey-flow/components/nodes/deploy-node";
import { OriginNode } from "./components/unkey-flow/components/nodes/origin-node";
import { DefaultNode } from "./components/unkey-flow/components/nodes/default-node";
import type { TreeNode } from "./components/unkey-flow/types";
import { LiveIndicator } from "./components/unkey-flow/components/overlay/live";
import { useState } from "react";
import {
  DEFAULT_TREE,
  DevTreeGenerator,
} from "./components/unkey-flow/components/overlay/dev-tree-generator";

export default function DeploymentDetailsPage() {
  const [generatedTree, setGeneratedTree] = useState<DeploymentNode | null>(
    DEFAULT_TREE
  );
  return (
    <InfiniteCanvas
      overlay={
        <>
          <LiveIndicator />
          <DevTreeGenerator onTreeGenerate={(tree) => setGeneratedTree(tree)} />
        </>
      }
    >
      <TreeLayout
        // biome-ignore lint/style/noNonNullAssertion: <explanation>
        data={generatedTree!}
        nodeSpacing={{ x: 25, y: 75 }}
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
                  // @ts-expect-error Will make it more typesafe soon
                  // biome-ignore lint/style/noNonNullAssertion: Will make it typesafe soon
                  flagCode={parent.metadata.flagCode!}
                />
              );
            default:
              return <DefaultNode node={node} />;
          }
        }}
        renderConnection={(from, to, parent, child, waypoints) => {
          const parentDirection = (parent as TreeNode).direction ?? "vertical";
          return (
            <TreeConnectionLine
              key={`${parent.id}-${child.id}`}
              from={from}
              to={to}
              waypoints={waypoints}
              horizontal={parentDirection === "horizontal"}
            />
          );
        }}
      />
    </InfiniteCanvas>
  );
}
