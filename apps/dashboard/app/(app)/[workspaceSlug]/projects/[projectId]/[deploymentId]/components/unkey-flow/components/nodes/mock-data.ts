import type { DeploymentNode } from "./types";

export const deploymentTree: DeploymentNode = {
  id: "ingress",
  label: "INTERNET",
  metadata: {
    type: "origin",
  },
  children: [
    {
      id: "us-east-1",
      label: "us-east-1",
      metadata: {
        type: "region",
        flagCode: "us", // Changed from flagComponent: <UsFlag />
        zones: 2,
        instances: 28,
        replicas: 2,
        power: 31,
        storage: "768mi",
        bandwidth: "1gb",
        latency: "2.4ms",
        status: "active",
        health: "healthy",
      },
      children: [
        {
          id: "us-east-1-gw-9h4k-1",
          label: "gw-9h4k",
          metadata: {
            type: "instance",
            description: "Instance replica",
            instances: 28,
            replicas: 2,
            power: "31%",
            storage: "768mi",
            latency: "2.4ms",
            status: "active",
            health: "healthy",
          },
        },
        {
          id: "us-east-1-gw-9h4k-1-123213",
          label: "gw-9hasdsad4k",
          metadata: {
            type: "instance",
            description: "Instance replica",
            replicas: 2,
            power: "38%",
            latency: "6ms",
            status: "active",
            health: "healthy",
          },
        },
      ],
    },
    {
      id: "ap-east-1",
      label: "ap-east-1",
      metadata: {
        type: "region",
        flagCode: "hk", // Changed
        zones: 1,
        instances: 24,
        replicas: 2,
        power: 27,
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
            instances: 24,
            replicas: 2,
            power: "35%",
            storage: "512mi",
            latency: "9ms",
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
        flagCode: "in", // Changed
        zones: 2,
        instances: 24,
        replicas: 2,
        power: 23,
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
        flagCode: "eu", // Changed
        zones: 1,
        instances: 32,
        replicas: 2,
        power: 31,
        storage: "1gi",
        bandwidth: "1gb",
        latency: "2.8ms",
        status: "active",
        health: "healthy",
      },
      children: [
        {
          id: "eu-west-1-gw-2m9p-2",
          label: "gw-2m9p",
          metadata: {
            type: "instance",
            description: "Instance replica",
            instances: 32,
            replicas: 2,
            power: "31%",
            latency: "5.8ms",
            status: "active",
            health: "healthy",
          },
        },
      ],
    },
  ],
};
