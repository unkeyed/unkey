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
import { DevTreeGenerator } from "./components/unkey-flow/components/simulate/tree-generate";
import type { TreeNode } from "./components/unkey-flow/types";
import { generateDeploymentTree } from "./components/unkey-flow/components/simulate/simulate";
import { useState } from "react";

const DEFAULT_TREE = generateDeploymentTree({
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

export default function DeploymentDetailsPage() {
  const [generatedTree, setGeneratedTree] = useState<DeploymentNode | null>(
    DEFAULT_TREE
  );

  return (
    <InfiniteCanvas
      overlay={
        process.env.NODE_ENV === "development" ? (
          <DevTreeGenerator
            onGenerate={(config) =>
              setGeneratedTree(generateDeploymentTree(config))
            }
            onReset={() => setGeneratedTree(DEFAULT_TREE)}
          />
        ) : undefined
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
