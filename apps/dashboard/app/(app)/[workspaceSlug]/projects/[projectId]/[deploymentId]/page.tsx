"use client";
import { trpc } from "@/lib/trpc/client";
import { useState } from "react";
import { useProject } from "../layout-provider";
import { TreeConnectionLine, TreeLayout } from "./components/unkey-flow";
import { InfiniteCanvas } from "./components/unkey-flow/components/canvas/infinite-canvas";
import { DefaultNode } from "./components/unkey-flow/components/nodes/default-node";
import { GatewayNode, RegionNode } from "./components/unkey-flow/components/nodes/deploy-node";
import { OriginNode } from "./components/unkey-flow/components/nodes/origin-node";
import type { DeploymentNode } from "./components/unkey-flow/components/nodes/types";
import { LiveIndicator } from "./components/unkey-flow/components/overlay/live";
import { NodeDetailsPanel } from "./components/unkey-flow/components/overlay/node-details-panel";
import { ProjectDetails } from "./components/unkey-flow/components/overlay/project-details";
import { InternalDevTreeGenerator } from "./components/unkey-flow/components/simulate/tree-generate";

export default function DeploymentDetailsPage() {
  const { projectId, liveDeploymentId } = useProject();
  const [generatedTree, setGeneratedTree] = useState<DeploymentNode | null>(null);
  const [selectedNode, setSelectedNode] = useState<DeploymentNode>();

  const { data: defaultTree, isLoading } = trpc.deploy.network.get.useQuery({
    deploymentId: liveDeploymentId,
  });

  const currentTree = generatedTree ?? defaultTree;

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-gray-11">Loading deployment...</div>
      </div>
    );
  }

  if (!currentTree) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-gray-11">No deployment data available</div>
      </div>
    );
  }

  return (
    <InfiniteCanvas
      overlay={
        <>
          <NodeDetailsPanel node={selectedNode} />
          <ProjectDetails projectId={projectId} />
          <LiveIndicator />
          {process.env.NODE_ENV === "development" && (
            <InternalDevTreeGenerator
              deploymentId={liveDeploymentId}
              onGenerate={setGeneratedTree}
              onReset={() => setGeneratedTree(null)}
            />
          )}
        </>
      }
    >
      <TreeLayout
        data={currentTree}
        nodeSpacing={{ x: 25, y: 75 }}
        onNodeClick={(node) => setSelectedNode(node)}
        renderNode={(node, _, parent) => {
          switch (node.metadata.type) {
            case "origin":
              return <OriginNode node={node} />;
            case "region":
              return (
                <RegionNode node={node as DeploymentNode & { metadata: { type: "region" } }} />
              );
            case "gateway":
              if (!parent?.id) {
                throw new Error("Gateway node requires parent region");
              }

              // biome-ignore lint/correctness/noSwitchDeclarations: <explanation>
              const parentMetadata = parent.metadata;
              if (parentMetadata.type !== "region") {
                throw new Error("Gateway parent must be a region node");
              }

              return (
                <GatewayNode
                  node={node as DeploymentNode & { metadata: { type: "gateway" } }}
                  flagCode={parentMetadata.flagCode}
                />
              );
            default:
              return <DefaultNode node={node} />;
          }
        }}
        renderConnection={(path, parent, child) => {
          return <TreeConnectionLine key={`${parent.id}-${child.id}`} path={path} />;
        }}
      />
    </InfiniteCanvas>
  );
}
