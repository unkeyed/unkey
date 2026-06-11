"use client";

import { useApiKeyAuthId } from "@/hooks/use-api-key-auth-id";
import { useSectionContext } from "@/hooks/use-section-context";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
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

  const links = (() => {
    switch (context.type) {
      case "workspace":
      case "identity":
      // Settings and authorization keep the top-level workspace nav in the
      // global sidebar; their sub-pages live in the SecondaryNav rail (see
      // settings/layout and authorization/layout).
      case "settings":
      case "authorization":
        return buildWorkspaceSections(slug, segments);
      case "project":
        return context.appId
          ? buildAppLinks(slug, context.projectId, context.appId, segments)
          : buildProjectLinks(slug, context.projectId, segments);
      case "api":
        return buildApiLinks(slug, context.apiId, keyAuthId, segments);
      case "namespace":
        return buildNamespaceLinks(slug, context.namespaceId, segments);
    }
  })();

  return <NavLinkList links={links} />;
}
