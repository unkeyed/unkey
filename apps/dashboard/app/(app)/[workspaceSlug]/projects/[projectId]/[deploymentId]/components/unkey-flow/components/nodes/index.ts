export * from "./gateway-node";
export * from "./instance-node";

export * from "./skeleton-node/skeleton-layout";
export * from "./skeleton-node/skeleton-node";

export * from "./origin-node";

export {
  isOriginNode,
  isRegionNode,
  isGatewayNode,
  isSkeletonNode,
  type DeploymentNode,
} from "./types";
