"use client";

import { warmDeploymentsTable } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider-queries";
import { useApiKeyAuthId } from "@/hooks/use-api-key-auth-id";
import { usePrefetchWorkspaceUsage } from "@/hooks/use-prefetch-workspace-usage";
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
import { routes } from "@/lib/navigation/routes";
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
  const prefetchUsage = usePrefetchWorkspaceUsage();

  const workspaceSections = (segs: string[]) =>
    projectsNav
      ? buildProjectsNavWorkspaceSections(slug, segs, user?.role === "admin")
      : buildWorkspaceSections(slug, segs);
  const projectLinks = projectsNav ? buildProjectsNavProjectLinks : buildProjectLinks;

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
      case "project": {
        const { projectId, appId } = context;
        return appId
          ? buildAppLinks(slug, projectId, appId, segments).map((link) =>
              link.key === "deployments"
                ? { ...link, onIntent: () => warmDeploymentsTable(projectId, appId) }
                : link,
            )
          : projectLinks(slug, projectId, segments);
      }
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
          : workspaceSections(segments);
    }
  })();

  const settingsHref = routes.settings.general({ workspaceSlug: slug });

  return (
    <NavLinkList
      links={links.map((link) =>
        link.href === settingsHref ? { ...link, onIntent: prefetchUsage } : link,
      )}
    />
  );
}
