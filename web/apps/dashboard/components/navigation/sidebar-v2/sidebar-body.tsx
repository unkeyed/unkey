"use client";

import { useApiKeyAuthId } from "@/hooks/use-api-key-auth-id";
import { useProjectSlug } from "@/hooks/use-route-slugs";
import { useSectionContext } from "@/hooks/use-section-context";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import {
  buildApiLinks,
  buildAppLinks,
  buildAuthorizationLinks,
  buildNamespaceLinks,
  buildProjectLinks,
  buildSettingsLinks,
  buildWorkspaceSections,
} from "@/lib/navigation/leaves";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
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

  const projectSlug = useProjectSlug();
  const appSlug = context.type === "project" ? context.appSlug : undefined;
  const appQuery = useLiveQuery(
    (q) =>
      projectSlug && appSlug
        ? q
            .from({ app: collection.apps })
            .where(({ app }) => and(eq(app.projectSlug, projectSlug), eq(app.slug, appSlug)))
        : undefined,
    [projectSlug, appSlug],
  );
  const appId = appQuery.data?.at(0)?.id;

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
