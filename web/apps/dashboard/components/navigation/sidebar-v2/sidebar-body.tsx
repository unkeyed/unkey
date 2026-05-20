"use client";

import { useApiKeyAuthId } from "@/hooks/use-api-key-auth-id";
import { useSectionContext } from "@/hooks/use-section-context";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import {
  buildApiLinks,
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
  const segments = useSelectedLayoutSegments().slice(1);
  const { slug } = useWorkspaceNavigation();
  const keyAuthId = useApiKeyAuthId(context.type === "api" ? context.apiId : undefined);

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
        return buildProjectLinks(slug, context.projectId, segments);
      case "api":
        return buildApiLinks(slug, context.apiId, keyAuthId, segments);
      case "namespace":
        return buildNamespaceLinks(slug, context.namespaceId, segments);
    }
  })();

  return <NavLinkList links={links} />;
}
