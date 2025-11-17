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

type RegionInfo = {
  name: string;
  location: string;
};

const REGION_INFO: Record<RegionMetadata["flagCode"], RegionInfo> = {
  us: { name: "US East", location: "N. Virginia" },
  de: { name: "EU Central", location: "Frankfurt" },
  au: { name: "AP Southeast", location: "Sydney" },
  jp: { name: "AP Northeast", location: "Tokyo" },
  in: { name: "AP South", location: "Mumbai" },
  br: { name: "SA East", location: "São Paulo" },
} as const;

export type {
  DeploymentNode,
  NodeMetadata,
  RegionMetadata,
  InstanceMetadata,
  OriginMetadata,
  HealthStatus,
  RegionInfo,
};

export { REGION_INFO };
