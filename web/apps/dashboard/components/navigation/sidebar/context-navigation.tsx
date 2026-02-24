"use client";

import { SidebarGroup, SidebarMenu } from "@/components/ui/sidebar";
import type { NavigationContext } from "@/hooks/use-navigation-context";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useSelectedLayoutSegments } from "next/navigation";
import { useEffect, useMemo } from "react";
import { NavItems } from "./app-sidebar/components/nav-items";
import { useApiKeyspace } from "./app-sidebar/hooks/use-api-keyspace";
import { useApiNavigation } from "./app-sidebar/hooks/use-api-navigation";
import { useNamespaceName } from "./app-sidebar/hooks/use-namespace-name";
import { useProjectData } from "./app-sidebar/hooks/use-project-data";
import { useProjectNavigation } from "./app-sidebar/hooks/use-projects-navigation";
import { useRatelimitNavigation } from "./app-sidebar/hooks/use-ratelimit-navigation";
import {
  createApiManagementNavigation,
  createApiNavigation,
  createDeployNavigation,
  createNamespaceNavigation,
  createProjectNavigation,
} from "./navigation-configs";

interface ContextNavigationProps {
  context: NavigationContext;
  onResourceNameFetched?: (name: string | undefined) => void;
}

/**
 * Context-aware navigation component that renders appropriate navigation items
 * based on the current context (product-level or resource-level).
 */
export function ContextNavigation({ context, onResourceNameFetched }: ContextNavigationProps) {
  const rawSegments = useSelectedLayoutSegments();
  const workspace = useWorkspaceNavigation();

  // Memoize segments to prevent unnecessary re-renders
  const segments = useMemo(() => rawSegments ?? [], [rawSegments]);

  // Generate base navigation items based on context
  const baseNavItems = useMemo(() => {
    if (context.type === "resource") {
      // Resource-level navigation
      switch (context.resourceType) {
        case "api":
          return createApiNavigation(context.resourceId, workspace, segments, context.keyAuthId);
        case "project":
          return createProjectNavigation(context.resourceId, workspace, segments);
        case "namespace":
          return createNamespaceNavigation(context.resourceId, workspace, segments);
      }
    }

    // Product-level navigation
    if (context.product === "deploy") {
      return createDeployNavigation(segments, workspace);
    }

    return createApiManagementNavigation(segments, workspace);
  }, [context, segments, workspace]);

  // Enhance navigation items with dynamic data (APIs, Projects, Namespaces lists)
  // These hooks only modify items if they find matching base items (e.g., /apis, /projects, /ratelimits)
  // At resource-level, these won't match anything and will return items unchanged
  const { enhancedNavItems: withApis } = useApiNavigation(baseNavItems);
  const { enhancedNavItems: withRatelimits } = useRatelimitNavigation(withApis);
  const { enhancedNavItems: withProjects } = useProjectNavigation(withRatelimits);

  // For API resources, enhance with keyspace ID and get API name
  const apiId =
    context.type === "resource" && context.resourceType === "api" ? context.resourceId : undefined;
  // Always fetch API data when we have an apiId to get the name and keyspace
  const { enhancedNavItems: withApiData, apiName } = useApiKeyspace(withProjects, apiId);

  // For project resources, enhance with project name
  const projectId =
    context.type === "resource" && context.resourceType === "project"
      ? context.resourceId
      : undefined;
  const { enhancedNavItems: finalNavItems, projectName } = useProjectData(withApiData, projectId);

  // For namespace resources, get namespace name
  const namespaceId =
    context.type === "resource" && context.resourceType === "namespace"
      ? context.resourceId
      : undefined;
  const namespaceName = useNamespaceName(namespaceId);

  // Notify parent when resource name is fetched
  useEffect(() => {
    if (onResourceNameFetched) {
      if (apiName) {
        onResourceNameFetched(apiName);
      } else if (projectName) {
        onResourceNameFetched(projectName);
      } else if (namespaceName) {
        onResourceNameFetched(namespaceName);
      }
    }
  }, [apiName, projectName, namespaceName, onResourceNameFetched]);

  return (
    <SidebarGroup>
      <SidebarMenu className="gap-2">
        {finalNavItems.map((item) => (
          <NavItems
            key={item.label as string}
            item={item}
            isResourceLevel={context.type === "resource"}
          />
        ))}
      </SidebarMenu>
    </SidebarGroup>
  );
}
