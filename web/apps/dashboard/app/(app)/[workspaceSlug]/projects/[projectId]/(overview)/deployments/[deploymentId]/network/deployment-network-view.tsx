"use client";
import { trpc } from "@/lib/trpc/client";
import { useCallback, useEffect, useRef, useState } from "react";
import { useDeployment } from "../layout-provider";
import {
  COLLAPSE_THRESHOLD,
  type DeploymentNode,
  InfiniteCanvas,
  InternalDevTreeGenerator,
  LiveIndicator,
  NodeDetailsPanel,
  type OriginNode as OriginNodeType,
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
import { InstanceNode } from "./unkey-flow/components/nodes/instance-node";
import { OriginNode } from "./unkey-flow/components/nodes/origin-node";

interface DeploymentNetworkViewProps {
  showProjectDetails?: boolean;
  showNodeDetails?: boolean;
}

export function DeploymentNetworkView({
  showProjectDetails = false,
  showNodeDetails = false,
}: DeploymentNetworkViewProps) {
  const { deployment } = useDeployment();
  const [generatedTree, setGeneratedTree] = useState<DeploymentNode | null>(null);
  const [selectedNode, setSelectedNode] = useState<DeploymentNode | null>(null);
  const [collapsedSentinelIds, setCollapsedSentinelIds] = useState<Set<string>>(new Set());
  const hasAutoCollapsed = useRef(false);

  const { data: defaultTree, isLoading } = trpc.deploy.network.get.useQuery(
    {
      deploymentId: deployment.id,
    },
    { refetchInterval: 2000 },
  );

  const currentTree = generatedTree ?? defaultTree ?? SKELETON_TREE;
  const isShowingSkeleton = isLoading && !generatedTree;

  const toggleSentinel = useCallback((id: string) => {
    setCollapsedSentinelIds((prev) => {
      const next = new Set(prev);
      next[next.has(id) ? "delete" : "add"](id);
      return next;
    });
  }, []);

  // Triggers auto collapse for sentinels with more than 3 children
  useEffect(() => {
    if (hasAutoCollapsed.current || !defaultTree) {
      return;
    }
    hasAutoCollapsed.current = true;

    const toCollapse = computeAutoCollapsedSentinels(defaultTree);
    if (toCollapse.size > 0) {
      setCollapsedSentinelIds(toCollapse);
    }
  }, [defaultTree]);

  const visibleTree = currentTree as OriginNodeType;

  const renderDeploymentNode = useCallback(
    (node: DeploymentNode, parent?: DeploymentNode): React.ReactNode => {
      if (isSkeletonNode(node)) {
        return <SkeletonNode />;
      }

      if (isOriginNode(node)) {
        return <OriginNode node={node} />;
      }

      if (isSentinelNode(node)) {
        return (
          <SentinelNode
            node={node}
            deploymentId={deployment.id}
            isCollapsed={collapsedSentinelIds.has(node.id)}
            onToggleCollapse={isShowingSkeleton ? undefined : () => toggleSentinel(node.id)}
          />
        );
      }

      if (isInstanceNode(node)) {
        if (!parent || !isSentinelNode(parent)) {
          throw new Error("Instance node requires parent sentinel");
        }
        if (collapsedSentinelIds.has(parent.id)) {
          return (
            <div className="invisible">
              <InstanceNode
                node={node}
                flagCode={parent.metadata.flagCode}
                deploymentId={deployment.id}
                stacked
              />
            </div>
          );
        }
        return (
          <InstanceNode
            node={node}
            flagCode={parent.metadata.flagCode}
            deploymentId={deployment.id}
          />
        );
      }

      // This will yell at you if you don't handle a node type
      const _exhaustive: never = node;
      return _exhaustive;
    },
    [deployment.id, collapsedSentinelIds, toggleSentinel, isShowingSkeleton],
  );

  return (
    <InfiniteCanvas
      defaultZoom={0.85}
      overlay={
        <>
          {showNodeDetails && (
            <NodeDetailsPanel node={selectedNode} onClose={() => setSelectedNode(null)} />
          )}

          {showProjectDetails && <ProjectDetails />}
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
        data={visibleTree}
        nodeSpacing={{ x: 75, y: 100 }}
        onNodeClick={isShowingSkeleton ? undefined : (node) => setSelectedNode(node)}
        renderNode={renderDeploymentNode}
        renderConnection={(path, parent, child) => {
          if (isSentinelNode(parent) && collapsedSentinelIds.has(parent.id)) {
            return null;
          }
          return <TreeConnectionLine key={`${parent.id}-${child.id}`} path={path} />;
        }}
      />
    </InfiniteCanvas>
  );
}

function computeAutoCollapsedSentinels(tree: OriginNodeType): Set<string> {
  const ids = (tree.children ?? [])
    // This is purely needed for proper type inference
    .filter(isSentinelNode)
    // If instance nodes exceeds threshold then we mark that sentinel as collapsed
    .filter((s) => (s.children ?? []).filter(isInstanceNode).length > COLLAPSE_THRESHOLD)
    .map((s) => s.id);

  return new Set(ids);
}
