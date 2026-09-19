"use client";

import { useApiKeyAuthId } from "@/hooks/use-api-key-auth-id";
import { useProject } from "@/hooks/use-project";
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
import { useWorkspace } from "@/providers/workspace-provider";
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
  const { user } = useWorkspace();
  const { project } = useProject();

  const workspaceSections = (segs: string[]) =>
    projectsNav
      ? buildProjectsNavWorkspaceSections(slug, segs, user?.role === "admin")
      : buildWorkspaceSections(slug, segs);
  const projectLinks = (segs: string[], projectId: string) =>
    projectsNav
      ? buildProjectsNavProjectLinks(slug, projectId, segs, { isDefault: project?.isDefault })
      : buildProjectLinks(slug, projectId, segs);

  const links = (() => {
    switch (context.type) {
      case "workspace":
      // Settings and Authorization keep the top-level workspace nav in the
      // global sidebar; their sub-pages live in a SecondaryNav rail (see the
      // settings/authorization layouts).
      case "account":
      case "settings":
      case "authorization":
        return workspaceSections(segments);
      case "project":
        return context.appId
          ? buildAppLinks(slug, context.projectId, context.appId, segments)
          : projectLinks(segments, context.projectId);
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
          ? projectLinks(segments, context.projectId)
          : workspaceSections(segments);
    }
  })();

  return <NavLinkList links={links} />;
}
