"use client";
import { InfiniteCanvas } from "./components";
import { CanvasNode } from "./components/canvas-node";
import { TreeLayout } from "./components/tree-layout";
import type { TreeNode } from "./components/types";

const complexTree: TreeNode = {
  id: "app",
  label: "Application",
  children: [
    {
      id: "auth",
      label: "Authentication",
      children: [
        { id: "jwt", label: "JWT Service" },
        { id: "oauth", label: "OAuth Provider" },
      ],
    },
    // {
    //   id: "api",
    //   label: "API Layer",
    //   children: [
    //     {
    //       id: "rest",
    //       label: "REST API",
    //       children: [
    //         { id: "users", label: "Users Endpoint" },
    //         { id: "posts", label: "Posts Endpoint" },
    //       ],
    //     },
    //     {
    //       id: "graphql",
    //       label: "GraphQL",
    //       children: [
    //         { id: "queries", label: "Queries" },
    //         { id: "mutations", label: "Mutations" },
    //       ],
    //     },
    //   ],
    // },
    // {
    //   id: "data",
    //   label: "Data Layer",
    //   children: [
    //     { id: "postgres", label: "PostgreSQL" },
    //     { id: "redis", label: "Redis Cache" },
    //     { id: "s3", label: "S3 Storage" },
    //   ],
    // },
  ],
};

export default function DeploymentDetailsPage() {
  return (
    <InfiniteCanvas>
      <TreeLayout
        data={complexTree}
        nodeSpacing={{ x: 200, y: 150 }}
        renderNode={(node, pos) => (
          <CanvasNode key={node.id} x={pos.x} y={pos.y}>
            <div className="bg-white rounded-lg shadow p-4">{node.label}</div>
          </CanvasNode>
        )}
      />
    </InfiniteCanvas>
  );
}
