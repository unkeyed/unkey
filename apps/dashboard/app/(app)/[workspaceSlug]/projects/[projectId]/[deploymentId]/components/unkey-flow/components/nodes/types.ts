type HealthStatus =
  | "normal"
  | "unstable"
  | "degraded"
  | "unhealthy"
  | "recovering"
  | "health_syncing"
  | "unknown"
  | "disabled";

type OriginMetadata = {
  type: "origin";
};

type RegionMetadata = {
  type: "region";
  flagCode: "us" | "de" | "au" | "jp" | "in" | "br";
  zones: number;
  instances: number;
  replicas: number;
  power: number;
  storage: string;
  bandwidth: string;
  latency: string;
  status: "active" | "inactive";
  health: HealthStatus;
};

type InstanceMetadata = {
  type: "instance";
  description: string;
  instances?: number;
  replicas: number;
  power: string;
  cpu?: string;
  memory?: string;
  storage?: string;
  latency: string;
  status: "active" | "inactive";
  health: HealthStatus;
};

type NodeMetadata = OriginMetadata | RegionMetadata | InstanceMetadata;

type DeploymentNode = {
  id: string;
  label: string;
  metadata: NodeMetadata;
  children?: DeploymentNode[];
};

export type {
  DeploymentNode,
  NodeMetadata,
  RegionMetadata,
  InstanceMetadata,
  OriginMetadata,
  HealthStatus,
};
