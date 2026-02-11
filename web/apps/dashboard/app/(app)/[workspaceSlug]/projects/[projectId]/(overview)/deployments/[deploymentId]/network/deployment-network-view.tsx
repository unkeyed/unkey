"use client";
import { trpc } from "@/lib/trpc/client";
import { useState } from "react";
import { useDeployment } from "../layout-provider";
import {
  type DeploymentNode,
  InfiniteCanvas,
  InstanceNode,
  InternalDevTreeGenerator,
  LiveIndicator,
  NodeDetailsPanel,
  OriginNode,
  ProjectDetails,
  SKELETON_TREE,
  SentinelNode,
  SkeletonNode,
  TreeConnectionLine,
  TreeLayout,
  isInstanceNode,
  isOriginNode,
  isSentinelNode,
  isSkeletonNode,
} from "./unkey-flow";

interface DeploymentNetworkViewProps {
  showProjectDetails?: boolean;
  showNodeDetails?: boolean;
}

export function DeploymentNetworkView({
  showProjectDetails = false,
  showNodeDetails = false,
}: DeploymentNetworkViewProps) {
  const { deploymentId } = useDeployment();
  const [generatedTree, setGeneratedTree] = useState<DeploymentNode | null>(null);
  const [selectedNode, setSelectedNode] = useState<DeploymentNode | null>(null);

  const { data: defaultTree, isLoading } = trpc.deploy.network.get.useQuery(
    {
      deploymentId: deploymentId ?? "",
    },
    { refetchInterval: 2000, enabled: Boolean(deploymentId) },
  );

  const currentTree = generatedTree ?? defaultTree ?? SKELETON_TREE;
  const isShowingSkeleton = isLoading && !generatedTree;

  return (
    <InfiniteCanvas
      defaultZoom={1}
      overlay={
        <>
          {showNodeDetails && (
            <NodeDetailsPanel node={selectedNode} onClose={() => setSelectedNode(null)} />
          )}

          {showProjectDetails && <ProjectDetails />}
          <LiveIndicator />
          {process.env.NODE_ENV === "development" && (
            <InternalDevTreeGenerator
              // biome-ignore lint/style/noNonNullAssertion: will be fixed later, when we actually implement tRPC logic
              deploymentId={deploymentId!}
              onGenerate={setGeneratedTree}
              onReset={() => setGeneratedTree(null)}
            />
          )}
        </>
      }
    >
      <TreeLayout
        data={currentTree}
        nodeSpacing={{ x: 10, y: 100 }}
        onNodeClick={isShowingSkeleton ? undefined : (node) => setSelectedNode(node)}
        renderNode={(node, parent) => renderDeploymentNode(node, parent, deploymentId ?? undefined)}
        renderConnection={(path, parent, child) => (
          <TreeConnectionLine key={`${parent.id}-${child.id}`} path={path} />
        )}
      />
    </InfiniteCanvas>
  );
}

// renderDeployment function does not narrow types without type guards.
function renderDeploymentNode(
  node: DeploymentNode,
  parent?: DeploymentNode,
  deploymentId?: string,
): React.ReactNode {
  if (isSkeletonNode(node)) {
    return <SkeletonNode />;
  }

  if (isOriginNode(node)) {
    return <OriginNode node={node} />;
  }

  if (isSentinelNode(node)) {
    return <SentinelNode node={node} deploymentId={deploymentId} />;
  }

  if (isInstanceNode(node)) {
    if (!parent || !isSentinelNode(parent)) {
      throw new Error("Instance node requires parent sentinel");
    }
    return (
      <InstanceNode node={node} flagCode={parent.metadata.flagCode} deploymentId={deploymentId} />
    );
  }

  // This will yell at you if you don't handle a node type
  const _exhaustive: never = node;
  return _exhaustive;
}
