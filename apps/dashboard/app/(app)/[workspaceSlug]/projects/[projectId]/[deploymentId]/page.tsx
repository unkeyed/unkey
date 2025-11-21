"use client";
import { trpc } from "@/lib/trpc/client";
import { useState } from "react";
import { useProject } from "../layout-provider";
import { TreeConnectionLine, TreeLayout } from "./components/unkey-flow";
import { InfiniteCanvas } from "./components/unkey-flow/components/canvas/infinite-canvas";
import {
  type DeploymentNode,
  GatewayNode,
  OriginNode,
  RegionNode,
  SKELETON_TREE,
  SkeletonNode,
  isGatewayNode,
  isOriginNode,
  isRegionNode,
  isSkeletonNode,
} from "./components/unkey-flow/components/nodes";
import { LiveIndicator } from "./components/unkey-flow/components/overlay/live";
import { NodeDetailsPanel } from "./components/unkey-flow/components/overlay/node-details-panel";
import { ProjectDetails } from "./components/unkey-flow/components/overlay/project-details";
import { InternalDevTreeGenerator } from "./components/unkey-flow/components/simulate/tree-generate";

export default function DeploymentDetailsPage() {
  const { projectId, liveDeploymentId } = useProject();
  const [generatedTree, setGeneratedTree] = useState<DeploymentNode | null>(null);
  const [selectedNode, setSelectedNode] = useState<DeploymentNode>();

  const { data: defaultTree, isLoading } = trpc.deploy.network.get.useQuery({
    // biome-ignore lint/style/noNonNullAssertion: will be fixed later, when we actually implement tRPC logic
    deploymentId: liveDeploymentId!,
  });

  const currentTree = generatedTree ?? defaultTree ?? SKELETON_TREE;
  const isShowingSkeleton = isLoading && !generatedTree;

  return (
    <InfiniteCanvas
      overlay={
        <>
          <NodeDetailsPanel node={selectedNode} />
          <ProjectDetails projectId={projectId} />
          <LiveIndicator />
          {process.env.NODE_ENV === "development" && (
            <InternalDevTreeGenerator
              // biome-ignore lint/style/noNonNullAssertion: will be fixed later, when we actually implement tRPC logic
              deploymentId={liveDeploymentId!}
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
        onNodeClick={isShowingSkeleton ? undefined : (node) => setSelectedNode(node)}
        renderNode={(node, parent) => renderDeploymentNode(node, parent)}
        renderConnection={(path, parent, child) => (
          <TreeConnectionLine key={`${parent.id}-${child.id}`} path={path} />
        )}
      />
    </InfiniteCanvas>
  );
}

// renderDeployment function does not narrow types without type guards.
function renderDeploymentNode(node: DeploymentNode, parent?: DeploymentNode): React.ReactNode {
  if (isSkeletonNode(node)) {
    return <SkeletonNode />;
  }

  if (isOriginNode(node)) {
    return <OriginNode node={node} />;
  }

  if (isRegionNode(node)) {
    return <RegionNode node={node} />;
  }

  if (isGatewayNode(node)) {
    if (!parent || !isRegionNode(parent)) {
      throw new Error("Gateway node requires parent region");
    }
    return <GatewayNode node={node} flagCode={parent.metadata.flagCode} />;
  }

  // This will yell at you if you don't handle a node type
  const _exhaustive: never = node;
  return _exhaustive;
}
