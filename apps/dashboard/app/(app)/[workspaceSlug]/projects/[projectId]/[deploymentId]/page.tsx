"use client";
import { InfiniteCanvas } from "./components";
import { CanvasNode } from "./components/canvas-node";
import { AnimatedConnectionLine, TreeLayout } from "./components/tree-layout";
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
        { id: "oauth-1", label: "OAuth Provider" },
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
    <InfiniteCanvas gridSize={25} gridDotSize={2} gridDotColor="#9ca3af">
      <TreeLayout
        data={complexTree}
        nodeSpacing={{ x: 100, y: 150 }}
        renderNode={(node) => (
          <div className="w-[500px] h-[70px] border border-grayA-4 rounded-[14px] bg-gray-2">
            {node.label}
          </div>
        )}
        renderConnection={(from, to, parent, child) => (
          <AnimatedConnectionLine
            key={`${parent.id}-${child.id}`}
            id={`${parent.id}-${child.id}`}
            from={{ x: from.x, y: from.y + 35 }} // bottom of parent (70/2)
            to={{ x: to.x, y: to.y - 35 }} // top of child (70/2)
          />
        )}
      />
    </InfiniteCanvas>
  );
}
