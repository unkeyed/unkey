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
  rps?: number;
  cpu?: number;
  memory?: number;
  storage?: number;
  latency: string;
  health: HealthStatus;
};

type GatewayMetadata = {
  type: "gateway";
  description: string;
  instances?: number;
  replicas: number;
  rps?: number;
  cpu?: number;
  memory?: number;
  storage?: number;
  latency: string;
  health: HealthStatus;
};

type NodeMetadata = OriginMetadata | RegionMetadata | GatewayMetadata | { type: "skeleton" };

type DeploymentNode = {
  id: string;
  label: string;
  direction?: "horizontal" | "vertical";
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
  GatewayMetadata as InstanceMetadata,
  OriginMetadata,
  HealthStatus,
  RegionInfo,
};

export { REGION_INFO };
