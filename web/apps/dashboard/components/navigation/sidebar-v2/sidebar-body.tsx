"use client";

import { useApiKeyAuthId } from "@/hooks/use-api-key-auth-id";
import { useSectionContext } from "@/hooks/use-section-context";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useFlag } from "@/lib/flags/provider";
import {
  buildApiLinks,
  buildAppLinks,
  buildNamespaceLinks,
  buildProjectLinks,
  buildWorkspaceSections,
} from "@/lib/navigation/leaves";
import {
  buildProjectLinks as buildProjectsNavProjectLinks,
  buildWorkspaceSections as buildProjectsNavWorkspaceSections,
} from "@/lib/navigation/leaves-projects";
import { useSelectedLayoutSegments } from "next/navigation";
import { NavLinkList } from "./nav-link-list";

export function SidebarBody() {
  const context = useSectionContext();
  // useSelectedLayoutSegments includes route groups like "(project)"; strip
  // them so the index-based page lookups in leaves.ts stay stable.
  const segments = useSelectedLayoutSegments()
    .slice(1)
    .filter((segment) => !segment.startsWith("("));
  const { slug } = useWorkspaceNavigation();
  const { keyAuthId } = useApiKeyAuthId(context.type === "api" ? context.apiId : undefined);
  const portalManagement = useFlag("portalManagement");
  const projectsNav = useFlag("projectsNav");

  const workspaceSections = projectsNav
    ? buildProjectsNavWorkspaceSections
    : buildWorkspaceSections;
  const projectLinks = projectsNav ? buildProjectsNavProjectLinks : buildProjectLinks;

  const links = (() => {
    switch (context.type) {
      case "workspace":
      // Settings and Authorization keep the top-level workspace nav in the
      // global sidebar; their sub-pages live in a SecondaryNav rail (see the
      // settings/authorization layouts).
      case "settings":
      case "authorization":
        return workspaceSections(slug, segments);
      case "project":
        return context.appId
          ? buildAppLinks(slug, context.projectId, context.appId, segments)
          : projectLinks(slug, context.projectId, segments);
      case "api":
        return buildApiLinks(
          { workspaceSlug: slug, apiId: context.apiId, projectId: context.projectId },
          keyAuthId,
          segments,
          portalManagement,
        );
      case "namespace":
        return buildNamespaceLinks(
          { workspaceSlug: slug, namespaceId: context.namespaceId, projectId: context.projectId },
          segments,
        );
      case "identity":
        return context.projectId
          ? projectLinks(slug, context.projectId, segments)
          : workspaceSections(slug, segments);
    }
  })();

  return <NavLinkList links={links} />;
}
