"use client";

import { useParams, useSelectedLayoutSegments } from "next/navigation";
import { useEffect, useMemo } from "react";
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

const STORAGE_KEY = "selected-product";

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

  // Detect the current product from URL and update localStorage
  const detectedProduct = useMemo(() => {
    const hasWorkspaceSlug = segments.length > 0 && segments[0];
    const productSegment = segments[1];

    if (!hasWorkspaceSlug || !productSegment) {
      return null;
    }

    // Deploy product segments: projects, domains, environment-variables
    const isDeploySegment =
      productSegment.startsWith("project") ||
      productSegment.startsWith("domain") ||
      productSegment.startsWith("environment-variable");

    if (isDeploySegment && workspace.betaFeatures?.deployments === true) {
      return "deploy";
    }

    // API Management product segments: apis, ratelimits, authorization, logs, identities
    const isApiManagementSegment =
      productSegment.startsWith("api") ||
      productSegment.startsWith("ratelimit") ||
      productSegment === "authorization" ||
      productSegment === "logs" ||
      productSegment.startsWith("identit");

    if (isApiManagementSegment) {
      return "api-management";
    }

    return null;
  }, [segments, workspace.betaFeatures?.deployments]);

  // Update localStorage when we detect a product from the URL
  useEffect(() => {
    if (detectedProduct && typeof window !== "undefined") {
      localStorage.setItem(STORAGE_KEY, detectedProduct);
    }
  }, [detectedProduct]);

  // Memoize the context to prevent unnecessary re-renders
  return useMemo(() => {
    // Detect resource-level context by checking for resource ID params
    if (params.apiId) {
      return {
        type: "resource",
        resourceType: "api",
        resourceId: params.apiId as string,
        resourceName: undefined,
        keyAuthId: params.keyAuthId as string | undefined,
      };
    }

    if (params.projectId) {
      return {
        type: "resource",
        resourceType: "project",
        resourceId: params.projectId as string,
        resourceName: undefined,
      };
    }

    if (params.namespaceId) {
      return {
        type: "resource",
        resourceType: "namespace",
        resourceId: params.namespaceId as string,
        resourceName: undefined,
      };
    }

    // For product-level routes, use detected product
    if (detectedProduct) {
      return { type: "product", product: detectedProduct };
    }

    // For workspace-level routes (settings, audit, etc.)
    // use the selected product from state (which reads from localStorage)
    return { type: "product", product: selectedProduct };
  }, [params, detectedProduct, selectedProduct]);
}
