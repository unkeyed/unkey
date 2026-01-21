type HealthStatus =
  | "normal"
  | "unstable"
  | "degraded"
  | "unhealthy"
  | "recovering"
  | "health_syncing"
  | "unknown"
  | "disabled";

type BaseNode = {
  id: string;
  label: string;
  direction?: "horizontal" | "vertical";
};

type BaseMetrics = {
  rps?: number;
  cpu?: number;
  memory?: number;
  storage?: number;
  latency: string;
  health: HealthStatus;
};

type OriginNode = BaseNode & {
  metadata: {
    type: "origin";
  };
  children?: DeploymentNode[];
};

type RegionNode = BaseNode & {
  metadata: {
    type: "region";
    flagCode: "us" | "de" | "au" | "jp" | "in" | "br";
    zones: number;
    instances: number;
    replicas: number;
  } & BaseMetrics;
  children?: SentinelNode[];
};

type SentinelNode = BaseNode & {
  metadata: {
    type: "sentinel";
    description: string;
    instances?: number;
    replicas: number;
  } & BaseMetrics;
};

type SkeletonNode = BaseNode & {
  metadata: {
    type: "skeleton";
  };
  children?: SkeletonNode[];
};

type DeploymentNode = OriginNode | RegionNode | SentinelNode | SkeletonNode;

function isOriginNode(node: DeploymentNode): node is OriginNode {
  return node.metadata.type === "origin";
}

function isRegionNode(node: DeploymentNode): node is RegionNode {
  return node.metadata.type === "region";
}

function isSentinelNode(node: DeploymentNode): node is SentinelNode {
  return node.metadata.type === "sentinel";
}

function isSkeletonNode(node: DeploymentNode): node is SkeletonNode {
  return node.metadata.type === "skeleton";
}

type RegionInfo = {
  name: string;
  location: string;
};

const REGION_INFO: Record<RegionNode["metadata"]["flagCode"], RegionInfo> = {
  us: { name: "US East", location: "N. Virginia" },
  de: { name: "EU Central", location: "Frankfurt" },
  au: { name: "AP Southeast", location: "Sydney" },
  jp: { name: "AP Northeast", location: "Tokyo" },
  in: { name: "AP South", location: "Mumbai" },
  br: { name: "SA East", location: "São Paulo" },
} as const;

const DEFAULT_NODE_WIDTH = 230;
type NodeSize = { width: number; height: number };
/**
 * Since our nodes are custom-made, we can optimize layout through static heights and widths.
 * If things change over time, we can either update this list or create a ResizeObserver to track changes dynamically.
 */
const NODE_SIZES: Record<DeploymentNode["metadata"]["type"], NodeSize> = {
  origin: { width: 70, height: 20 },
  region: { width: DEFAULT_NODE_WIDTH, height: 70 },
  sentinel: { width: DEFAULT_NODE_WIDTH, height: 70 },
  skeleton: { width: DEFAULT_NODE_WIDTH, height: 70 },
} as const;

export type {
  DeploymentNode,
  OriginNode,
  RegionNode,
  SentinelNode,
  SkeletonNode,
  HealthStatus,
  RegionInfo,
  BaseMetrics,
};

export {
  isOriginNode,
  isRegionNode,
  isSentinelNode,
  isSkeletonNode,
  DEFAULT_NODE_WIDTH,
  REGION_INFO,
  NODE_SIZES,
};
