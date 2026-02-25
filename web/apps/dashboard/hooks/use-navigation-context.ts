"use client";

import { useParams, useSelectedLayoutSegments } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import { getCurrentProduct } from "./use-product-selection";
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
const STORAGE_EVENT = "product-selection-changed";

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
  const workspace = useWorkspaceNavigation();

  // Track localStorage changes via custom event
  const [storageVersion, setStorageVersion] = useState(0);

  useEffect(() => {
    const handleStorageChange = () => {
      setStorageVersion((v) => v + 1);
    };

    window.addEventListener(STORAGE_EVENT, handleStorageChange);
    return () => window.removeEventListener(STORAGE_EVENT, handleStorageChange);
  }, []);

  // Detect the current product from URL
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

  // Update localStorage ONLY when we detect a product from the URL
  // This ensures we don't change the product context when it's ambiguous
  useEffect(() => {
    if (detectedProduct && typeof window !== "undefined") {
      localStorage.setItem(STORAGE_KEY, detectedProduct);
    }
  }, [detectedProduct]);

  // Memoize the context to prevent unnecessary re-renders
  // Include storageVersion to re-compute when localStorage changes
  // biome-ignore lint/correctness/useExhaustiveDependencies: storageVersion forces re-computation on localStorage changes
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

    // For product-level routes, use detected product if available
    if (detectedProduct) {
      return { type: "product", product: detectedProduct };
    }

    // For workspace-level routes (settings, audit, etc.) where we can't determine the product,
    // fall back to the current product selection from localStorage
    // This ensures we don't change the product context when it's ambiguous
    const currentProduct = getCurrentProduct();
    return { type: "product", product: currentProduct };
  }, [params, detectedProduct, storageVersion]);
}
