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
  RegionNode,
  SKELETON_TREE,
  SkeletonNode,
  TreeConnectionLine,
  TreeLayout,
  isInstanceNode,
  isOriginNode,
  isRegionNode,
  isSkeletonNode,
} from "./unkey-flow";

interface DeploymentNetworkViewProps {
  showNodeDetails?: boolean;
}

export function DeploymentNetworkView({ showNodeDetails = false }: DeploymentNetworkViewProps) {
  const { deployment } = useDeployment();
  const [generatedTree, setGeneratedTree] = useState<DeploymentNode | null>(null);
  const [selectedNode, setSelectedNode] = useState<DeploymentNode | null>(null);

  const { data: defaultTree, isLoading } = trpc.deploy.network.get.useQuery(
    {
      deploymentId: deployment.id,
    },
    { refetchInterval: 2000 },
  );

  // One query for every region and instance card on the canvas. Asking per
  // card made the number of ClickHouse and deployment lookups scale with the
  // number of nodes a deployment runs.
  const { data: rps } = trpc.deploy.network.getRps.useQuery(
    {
      deploymentId: deployment.id,
    },
    { refetchInterval: 5000 },
  );

  const currentTree = generatedTree ?? defaultTree ?? SKELETON_TREE;
  const isShowingSkeleton = isLoading && !generatedTree;

  return (
    <InfiniteCanvas
      defaultZoom={0.85}
      overlay={
        <>
          {showNodeDetails && (
            <NodeDetailsPanel
              node={selectedNode}
              deploymentId={deployment.id}
              onClose={() => setSelectedNode(null)}
            />
          )}

          <LiveIndicator />
          {process.env.NODE_ENV === "development" && (
            <InternalDevTreeGenerator
              deploymentId={deployment.id}
              onGenerate={setGeneratedTree}
              onReset={() => setGeneratedTree(null)}
            />
          )}
        </>
      }
    >
      <TreeLayout
        data={currentTree}
        nodeSpacing={{ x: 20, y: 50 }}
        layoutConfig={{
          layout: { verticalOffset: -15, verticalSiblingSpacing: 0.5, horizontalIndent: 35 },
        }}
        onNodeClick={isShowingSkeleton ? undefined : (node) => setSelectedNode(node)}
        renderNode={(node, parent) => renderDeploymentNode(node, parent, rps)}
        renderConnection={(path, parent, child) => (
          <TreeConnectionLine key={`${parent.id}-${child.id}`} path={path} />
        )}
      />
    </InfiniteCanvas>
  );
}

type NetworkRps = {
  regions: Record<string, number>;
  instances: Record<string, number>;
};

// renderDeployment function does not narrow types without type guards.
function renderDeploymentNode(
  node: DeploymentNode,
  parent?: DeploymentNode,
  rps?: NetworkRps,
): React.ReactNode {
  if (isSkeletonNode(node)) {
    return <SkeletonNode />;
  }

  if (isOriginNode(node)) {
    return <OriginNode node={node} />;
  }

  if (isRegionNode(node)) {
    return <RegionNode node={node} rps={rps?.regions[node.label]} />;
  }

  if (isInstanceNode(node)) {
    if (!parent || !isRegionNode(parent)) {
      throw new Error("Instance node requires parent region");
    }
    return (
      <InstanceNode node={node} flagCode={parent.metadata.flagCode} rps={rps?.instances[node.id]} />
    );
  }

  // This will yell at you if you don't handle a node type
  const _exhaustive: never = node;
  return _exhaustive;
}
