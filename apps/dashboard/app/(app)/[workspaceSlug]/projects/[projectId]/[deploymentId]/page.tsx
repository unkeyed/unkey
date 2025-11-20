"use client";
import { trpc } from "@/lib/trpc/client";
import { useState } from "react";
import { useProject } from "../layout-provider";
import { TreeConnectionLine, TreeLayout } from "./components/unkey-flow";
import { InfiniteCanvas } from "./components/unkey-flow/components/canvas/infinite-canvas";
import { DefaultNode } from "./components/unkey-flow/components/nodes/default-node";
import {
  GatewayNode,
  RegionNode,
  SkeletonNode,
} from "./components/unkey-flow/components/nodes/deploy-node";
import { OriginNode } from "./components/unkey-flow/components/nodes/origin-node";
import type { DeploymentNode } from "./components/unkey-flow/components/nodes/types";
import { LiveIndicator } from "./components/unkey-flow/components/overlay/live";
import { NodeDetailsPanel } from "./components/unkey-flow/components/overlay/node-details-panel";
import { ProjectDetails } from "./components/unkey-flow/components/overlay/project-details";
import { InternalDevTreeGenerator } from "./components/unkey-flow/components/simulate/tree-generate";

const SKELETON_TREE: DeploymentNode = {
  id: "internet",
  label: "INTERNET",
  metadata: { type: "origin" },
  children: [
    {
      id: "us-east-1-skeleton",
      label: "us-east-1",
      direction: "horizontal",
      metadata: { type: "skeleton" } as const,
      children: [
        {
          id: "us-east-1-gw-1-skeleton",
          label: "gw-skeleton-1",
          metadata: { type: "skeleton" } as const,
        },
        {
          id: "us-east-1-gw-2-skeleton",
          label: "gw-skeleton-2",
          metadata: { type: "skeleton" } as const,
        },
      ],
    },
    {
      id: "eu-central-1-skeleton",
      label: "eu-central-1",
      direction: "horizontal",
      metadata: { type: "skeleton" } as const,
      children: [
        {
          id: "eu-central-1-gw-1-skeleton",
          label: "gw-skeleton-1",
          metadata: { type: "skeleton" } as const,
        },
        {
          id: "eu-central-1-gw-2-skeleton",
          label: "gw-skeleton-2",
          metadata: { type: "skeleton" } as const,
        },
        {
          id: "eu-central-1-gw-3-skeleton",
          label: "gw-skeleton-3",
          metadata: { type: "skeleton" } as const,
        },
      ],
    },
    {
      id: "ap-southeast-2-skeleton",
      label: "ap-southeast-2",
      direction: "horizontal",
      metadata: { type: "skeleton" } as const,
      children: [
        {
          id: "ap-southeast-2-gw-1-skeleton",
          label: "gw-skeleton-1",
          metadata: { type: "skeleton" } as const,
        },
        {
          id: "ap-southeast-2-gw-2-skeleton",
          label: "gw-skeleton-2",
          metadata: { type: "skeleton" } as const,
        },
      ],
    },
  ],
};

export default function DeploymentDetailsPage() {
  const { projectId, liveDeploymentId } = useProject();
  const [generatedTree, setGeneratedTree] = useState<DeploymentNode | null>(null);
  const [selectedNode, setSelectedNode] = useState<DeploymentNode>();

  const { data: defaultTree, isLoading } = trpc.deploy.network.get.useQuery({
    deploymentId: liveDeploymentId,
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
        onNodeClick={isShowingSkeleton ? undefined : (node) => setSelectedNode(node)}
        renderNode={(node, _, parent) => {
          if (node.metadata.type === "skeleton") {
            return <SkeletonNode />;
          }

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
