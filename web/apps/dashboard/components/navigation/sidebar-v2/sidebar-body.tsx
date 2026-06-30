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
  const keyAuthId = useApiKeyAuthId(context.type === "api" ? context.apiId : undefined);
  const appOverview = useFlag("appOverview");
  const portalManagement = useFlag("portalManagement");

  const links = (() => {
    switch (context.type) {
      case "workspace":
      case "identity":
      // Settings and Authorization keep the top-level workspace nav in the
      // global sidebar; their sub-pages live in a SecondaryNav rail (see the
      // settings/authorization layouts).
      case "settings":
      case "authorization":
        return buildWorkspaceSections(slug, segments, portalManagement);
      case "project":
        return context.appId
          ? buildAppLinks(slug, context.projectId, context.appId, segments, appOverview)
          : buildProjectLinks(slug, context.projectId, segments);
      case "api":
        return buildApiLinks(slug, context.apiId, keyAuthId, segments);
      case "namespace":
        return buildNamespaceLinks(slug, context.namespaceId, segments);
    }
  })();

  return <NavLinkList links={links} />;
}
