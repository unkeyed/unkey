"use client";

import { useParams, useSelectedLayoutSegments } from "next/navigation";
import { useMemo } from "react";
import { useProductSelection } from "./use-product-selection";
import { useWorkspaceNavigation } from "./use-workspace-navigation";

export type NavigationContext =
  | { type: "product"; product: "api-management" | "deploy" }
  | {
      type: "resource";
      resourceType: "api" | "project" | "namespace";
      resourceId: string;
      resourceName?: string;
      keyAuthId?: string; // For API resources, the keyspace/keyAuth ID
    };

/**
 * Hook to detect the current navigation context based on route params and segments.
 * Extracts resource name from URL segments when available.
 * Uses product selection state to maintain context on workspace-level routes.
 * Respects workspace feature flags - won't detect Deploy product if feature is disabled.
 *
 * Returns either:
 * - Product-level context: User is viewing a product (API Management or Deploy)
 * - Resource-level context: User is viewing a specific resource (API, Project, or Namespace)
 */
export function useNavigationContext(): NavigationContext {
  const segments = useSelectedLayoutSegments();
  const params = useParams();
  const { product: selectedProduct } = useProductSelection();
  const workspace = useWorkspaceNavigation();

  // Memoize the context to prevent unnecessary re-renders
  return useMemo(() => {
    // Detect resource-level context by checking for resource ID params
    if (params.apiId) {
      return {
        type: "resource",
        resourceType: "api",
        resourceId: params.apiId as string,
        // Resource name not available in URL segments for APIs
        resourceName: undefined,
        keyAuthId: params.keyAuthId as string | undefined,
      };
    }

    if (params.projectId) {
      return {
        type: "resource",
        resourceType: "project",
        resourceId: params.projectId as string,
        // Resource name not available in URL segments for projects
        resourceName: undefined,
      };
    }

    if (params.namespaceId) {
      return {
        type: "resource",
        resourceType: "namespace",
        resourceId: params.namespaceId as string,
        // Resource name not available in URL segments for namespaces
        resourceName: undefined,
      };
    }

    // Detect product-level context based on route segments
    // Only detect from URL if we have a workspace slug (segments[0] exists)
    // and we're in a product-specific route (segments[1] exists)
    const hasWorkspaceSlug = segments.length > 0 && segments[0];
    const productSegment = segments[1];

    if (hasWorkspaceSlug && productSegment) {
      // Deploy product segments: projects, domains, environment-variables
      // Check if the segment STARTS WITH these keywords to handle 404s like /domainsdomains
      // Only detect Deploy if the workspace has the feature enabled
      const isDeploySegment =
        productSegment.startsWith("project") ||
        productSegment.startsWith("domain") ||
        productSegment.startsWith("environment-variable");

      if (isDeploySegment && workspace.betaFeatures?.deployments === true) {
        return { type: "product", product: "deploy" };
      }

      // API Management product segments: apis, ratelimits, authorization, logs, identities
      // Check if the segment STARTS WITH these keywords to handle 404s
      const isApiManagementSegment =
        productSegment.startsWith("api") ||
        productSegment.startsWith("ratelimit") ||
        productSegment === "authorization" ||
        productSegment === "logs" ||
        productSegment.startsWith("identit");

      if (isApiManagementSegment) {
        return { type: "product", product: "api-management" };
      }
    }

    // For workspace-level routes (settings, audit, etc.)
    // or routes that don't match any product pattern, use the selected product from state
    return { type: "product", product: selectedProduct };
  }, [params, segments, selectedProduct, workspace.betaFeatures?.deployments]);
}
