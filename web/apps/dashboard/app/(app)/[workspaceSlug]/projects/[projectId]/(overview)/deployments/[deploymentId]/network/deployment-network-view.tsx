"use client";
import { trpc } from "@/lib/trpc/client";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useDeployment } from "../layout-provider";
import {
  COLLAPSE_THRESHOLD,
  DEFAULT_NODE_HEIGHT,
  DEFAULT_NODE_WIDTH,
  type DeploymentNode,
  InfiniteCanvas,
  type InstanceNode as InstanceNodeType,
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
import { InstanceNode } from "./unkey-flow/components/nodes/instance-node";

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
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }, []);

  useEffect(() => {
    if (hasAutoCollapsed.current || !defaultTree) {
      return;
    }
    hasAutoCollapsed.current = true;
    const toCollapse = new Set<string>();
    if (isOriginNode(defaultTree)) {
      for (const child of defaultTree.children ?? []) {
        if (
          isSentinelNode(child) &&
          (child.children ?? []).filter(isInstanceNode).length > COLLAPSE_THRESHOLD
        ) {
          toCollapse.add(child.id);
        }
      }
    }
    if (toCollapse.size > 0) {
      setCollapsedSentinelIds(toCollapse);
    }
  }, [defaultTree]);

  const { visibleTree, sentinelChildrenMap } = useMemo(() => {
    const map = new Map<string, InstanceNodeType[]>();

    function collapse(node: DeploymentNode): DeploymentNode {
      if (isOriginNode(node)) {
        return { ...node, children: node.children?.map((c) => collapse(c)) };
      }
      if (isSentinelNode(node)) {
        map.set(node.id, (node.children ?? []).filter(isInstanceNode));
        return collapsedSentinelIds.has(node.id)
          ? { ...node, children: node.children?.slice(0, 1) }
          : node;
      }
      return node;
    }

    return { visibleTree: collapse(currentTree), sentinelChildrenMap: map };
  }, [currentTree, collapsedSentinelIds]);

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
          const instances = sentinelChildrenMap.get(parent.id) ?? [];
          const totalLayers = instances.length;
          const step = 10;
          const frontOffset = (totalLayers - 1) * step;
          // pointer-events-none: stacked instances are not individually interactive
          // users must expand the sentinel first via its toggle button
          return (
            <div
              className="relative pointer-events-none"
              style={{
                height: frontOffset + DEFAULT_NODE_HEIGHT,
                width: frontOffset + DEFAULT_NODE_WIDTH,
              }}
            >
              {instances
                .slice(1)
                .reverse()
                .map((inst, i) => (
                  <div key={inst.id} className="absolute" style={{ top: i * step, left: i * step }}>
                    <InstanceNode
                      node={inst}
                      flagCode={parent.metadata.flagCode}
                      deploymentId={deployment.id}
                      stacked
                    />
                  </div>
                ))}
              <div className="absolute" style={{ top: frontOffset, left: frontOffset }}>
                <InstanceNode
                  node={node}
                  flagCode={parent.metadata.flagCode}
                  deploymentId={deployment.id}
                  stacked
                />
              </div>
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
    [deployment.id, collapsedSentinelIds, toggleSentinel, isShowingSkeleton, sentinelChildrenMap],
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
        renderConnection={(path, parent, child) => (
          <TreeConnectionLine key={`${parent.id}-${child.id}`} path={path} />
        )}
      />
    </InfiniteCanvas>
  );
}
