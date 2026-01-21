"use client";
import { trpc } from "@/lib/trpc/client";
import { useState } from "react";
import {
  type DeploymentNode,
  InfiniteCanvas,
  InternalDevTreeGenerator,
  LiveIndicator,
  NodeDetailsPanel,
  OriginNode,
  ProjectDetails,
  RegionNode,
  SKELETON_TREE,
  SentinelNode,
  SkeletonNode,
  TreeConnectionLine,
  TreeLayout,
  isOriginNode,
  isRegionNode,
  isSentinelNode,
  isSkeletonNode,
} from "./unkey-flow";
import { useProject } from "../../(overview)/layout-provider";

export default function DeploymentDetailsPage() {
  const { projectId, liveDeploymentId } = useProject();
  const [generatedTree, setGeneratedTree] = useState<DeploymentNode | null>(null);
  const [selectedNode, setSelectedNode] = useState<DeploymentNode | null>(null);

  const { data: defaultTree, isLoading } = trpc.deploy.network.get.useQuery();

  const currentTree = generatedTree ?? defaultTree ?? SKELETON_TREE;
  const isShowingSkeleton = isLoading && !generatedTree;

  return (
    <InfiniteCanvas
      overlay={
        <>
          <NodeDetailsPanel node={selectedNode} onClose={() => setSelectedNode(null)} />
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

  if (isSentinelNode(node)) {
    if (!parent || !isRegionNode(parent)) {
      throw new Error("Sentinel node requires parent region");
    }
    return <SentinelNode node={node} flagCode={parent.metadata.flagCode} />;
  }

  // This will yell at you if you don't handle a node type
  const _exhaustive: never = node;
  return _exhaustive;
}
