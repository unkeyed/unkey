"use client";

import { readLastProjectId } from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/store";
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
import { useEffect, useState } from "react";
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

  // Authorization and identities are project-scoped concepts living at
  // workspace URLs, so they keep the sidebar of the project the user was last
  // inside. Resolved in an effect (localStorage) so SSR stays consistent.
  const [lastProjectId, setLastProjectId] = useState<string | null>(null);
  const keepProjectContext = context.type === "authorization" || context.type === "identity";
  useEffect(() => {
    if (keepProjectContext) {
      setLastProjectId(readLastProjectId());
    }
  }, [keepProjectContext]);

  const links = (() => {
    switch (context.type) {
      case "authorization":
      case "identity":
        if (lastProjectId) {
          return buildProjectLinks(
            slug,
            lastProjectId,
            [
              "projects",
              lastProjectId,
              context.type === "identity" ? "identities" : "authorization",
            ],
            portalManagement,
          );
        }
        return buildWorkspaceSections(slug, segments);
      case "workspace":
      // Settings keeps the top-level workspace nav in the global sidebar; its
      // sub-pages live in a SecondaryNav rail (see the settings layout).
      case "settings":
        return buildWorkspaceSections(slug, segments);
      case "project":
        return context.appId
          ? buildAppLinks(slug, context.projectId, context.appId, segments, appOverview)
          : buildProjectLinks(slug, context.projectId, segments, portalManagement);
      case "api":
        return buildApiLinks(slug, context.apiId, keyAuthId, segments, portalManagement);
      case "namespace":
        return buildNamespaceLinks(slug, context.namespaceId, segments);
    }
  })();

  return <NavLinkList links={links} />;
}
