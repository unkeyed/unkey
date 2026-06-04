"use client";

import { useApiKeyAuthId } from "@/hooks/use-api-key-auth-id";
import { useResolvedApp } from "@/hooks/use-resolved-project";
import { useSectionContext } from "@/hooks/use-section-context";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import {
  buildApiLinks,
  buildAppLinks,
  buildAuthorizationLinks,
  buildNamespaceLinks,
  buildProjectLinks,
  buildSettingsLinks,
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
  const { appId } = useResolvedApp(context.type === "project" ? context.appSlug : undefined);

  const links = (() => {
    switch (context.type) {
      case "workspace":
      case "identity":
        return buildWorkspaceSections(slug, segments);
      case "settings":
        return buildSettingsLinks(slug, segments);
      case "authorization":
        return buildAuthorizationLinks(slug, segments);
      case "project":
        return context.appSlug
          ? buildAppLinks(slug, context.projectSlug, context.appSlug, appId, segments)
          : buildProjectLinks(slug, context.projectSlug, segments);
      case "api":
        return buildApiLinks(slug, context.apiId, keyAuthId, segments);
      case "namespace":
        return buildNamespaceLinks(slug, context.namespaceId, segments);
    }
  })();

  return <NavLinkList links={links} />;
}
