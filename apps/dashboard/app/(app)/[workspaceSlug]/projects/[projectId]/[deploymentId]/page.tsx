"use client";
import { InfiniteCanvas } from "./components";
import { AnimatedConnectionLine, TreeLayout } from "./components/tree-layout";
import type { TreeNode } from "./components/types";

const deploymentTree: TreeNode = {
  id: "ingress",
  label: "Ingress",
  metadata: {
    type: "gateway",
    description: "Global entry layer",
    regions: 3,
    instances: 64,
    replicas: 2,
    latency: "3.1ms",
    status: "active",
    health: "healthy",
  },
  children: [
    {
      id: "ap-east-1",
      label: "ap-east-1",
      metadata: {
        type: "region",
        description: "2 availability zones",
        zones: 2,
        instances: 24,
        replicas: 2,
        power: 2,
        storage: "512mi",
        bandwidth: "1gb",
        latency: "3.1ms",
        status: "active",
        health: "healthy",
      },
      children: [
        {
          id: "ap-east-1-gw-7f2c-1",
          label: "gw-7f2c",
          metadata: {
            type: "instance",
            description: "Instance replica",
            replicas: 2,
            power: "35%",
            cpu: "44%",
            memory: "22%",
            latency: "9ms",
            status: "active",
            health: "healthy",
          },
        },
        {
          id: "ap-east-1-gw-7f2c-2",
          label: "gw-7f2c",
          metadata: {
            type: "instance",
            description: "Instance replica",
            instances: 24,
            replicas: 2,
            power: "23%",
            cpu: "37%",
            memory: "17%",
            latency: "4.51ms",
            status: "active",
            health: "healthy",
          },
        },
      ],
    },
    {
      id: "ap-south-1",
      label: "ap-south-1",
      metadata: {
        type: "region",
        description: "2 availability zones",
        zones: 3,
        instances: 24,
        replicas: 2,
        power: 2,
        storage: "512mi",
        bandwidth: "1gb",
        latency: "3.1ms",
        status: "active",
        health: "healthy",
      },
      children: [
        {
          id: "ap-south-1-gw-8k3d-1",
          label: "gw-8k3d",
          metadata: {
            type: "instance",
            description: "Instance replica",
            replicas: 2,
            power: "41%",
            cpu: "52%",
            memory: "28%",
            latency: "7ms",
            status: "active",
            health: "healthy",
          },
        },
        {
          id: "ap-south-1-gw-8k3d-2",
          label: "gw-8k3d",
          metadata: {
            type: "instance",
            description: "Instance replica",
            instances: 24,
            replicas: 2,
            power: "19%",
            cpu: "31%",
            memory: "15%",
            latency: "5.2ms",
            status: "active",
            health: "healthy",
          },
        },
      ],
    },
    {
      id: "eu-west-1",
      label: "eu-west-1",
      metadata: {
        type: "region",
        description: "3 availability zones",
        zones: 3,
        instances: 32,
        replicas: 2,
        power: 3,
        storage: "1gi",
        bandwidth: "1gb",
        latency: "2.8ms",
        status: "active",
        health: "healthy",
      },
      children: [
        {
          id: "eu-west-1-gw-2m9p-1",
          label: "gw-2m9p",
          metadata: {
            type: "instance",
            description: "Instance replica",
            replicas: 2,
            power: "28%",
            cpu: "39%",
            memory: "19%",
            latency: "6ms",
            status: "active",
            health: "healthy",
          },
        },
        {
          id: "eu-west-1-gw-2m9p-2",
          label: "gw-2m9p",
          metadata: {
            type: "instance",
            description: "Instance replica",
            instances: 32,
            replicas: 2,
            power: "31%",
            cpu: "45%",
            memory: "21%",
            latency: "5.8ms",
            status: "active",
            health: "healthy",
          },
        },
      ],
    },
  ],
};
export default function DeploymentDetailsPage() {
  return (
    <InfiniteCanvas gridSize={35} gridDotSize={2.5} gridDotColor="#E5E5EA">
      <TreeLayout
        data={deploymentTree}
        nodeSpacing={{ x: 25, y: 130 }}
        renderNode={(node) => (
          <div className="w-[500px] h-[70px] border border-grayA-4 rounded-[14px] bg-gray-1 text-center">
            {node.label}
          </div>
        )}
        renderConnection={(from, to, parent, child) => {
          const childIndex =
            parent.children?.findIndex((c) => c.id === child.id) ?? 0;
          const childCount = parent.children?.length ?? 1;

          // Spread children horizontally across parent width
          const xOffset = (childIndex - (childCount - 1) / 2) * 5;

          return (
            <AnimatedConnectionLine
              key={`${parent.id}-${child.id}`}
              from={{ x: from.x + xOffset, y: from.y + 35 }}
              to={{ x: to.x, y: to.y - 35 }}
            />
          );
        }}
      />

      <foreignObject x={-400} y={-200} width={300} height={150}>
        <div className="bg-white dark:bg-gray-9 border border-gray-3 dark:border-gray-7 rounded-lg p-4 shadow-lg">
          <h3 className="text-lg font-semibold mb-2">System Stats</h3>
          <p className="text-sm text-gray-6 dark:text-gray-4">
            Total Instances: 80
          </p>
          <p className="text-sm text-gray-6 dark:text-gray-4">
            Avg Latency: 4.2ms
          </p>
        </div>
      </foreignObject>

      <foreignObject x={600} y={-100} width={200} height={100}>
        <div className="bg-success-5 dark:bg-success-9 border border-success-3 dark:border-success-7 rounded-lg p-3">
          <p className="text-sm font-medium text-success-8 dark:text-success-2">
            All systems operational
          </p>
        </div>
      </foreignObject>
    </InfiniteCanvas>
  );
}
