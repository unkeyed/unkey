"use client";

import { useParams, useSelectedLayoutSegments } from "next/navigation";

export type NavigationContext =
  | { type: "product"; product: "api-management" | "deploy" }
  | {
      type: "resource";
      resourceType: "api" | "project" | "namespace";
      resourceId: string;
    };

/**
 * Hook to detect the current navigation context based on route params and segments.
 *
 * Returns either:
 * - Product-level context: User is viewing a product (API Management or Deploy)
 * - Resource-level context: User is viewing a specific resource (API, Project, or Namespace)
 */
export function useNavigationContext(): NavigationContext {
  const segments = useSelectedLayoutSegments();
  const params = useParams();

  // Detect resource-level context by checking for resource ID params
  if (params.apiId) {
    return {
      type: "resource",
      resourceType: "api",
      resourceId: params.apiId as string,
    };
  }

  if (params.projectId) {
    return {
      type: "resource",
      resourceType: "project",
      resourceId: params.projectId as string,
    };
  }

  if (params.namespaceId) {
    return {
      type: "resource",
      resourceType: "namespace",
      resourceId: params.namespaceId as string,
    };
  }

  // Detect product-level context based on route segments
  // Deploy product segments: projects, domains, environment-variables
  const firstSegment = segments[1];
  if (["projects", "domains", "environment-variables"].includes(firstSegment)) {
    return { type: "product", product: "deploy" };
  }

  // Default to API Management product
  return { type: "product", product: "api-management" };
}
