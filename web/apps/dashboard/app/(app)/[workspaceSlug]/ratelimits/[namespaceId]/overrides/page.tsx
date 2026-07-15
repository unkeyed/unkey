"use client";
import { use } from "react";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { NamespaceNavbar } from "../namespace-navbar";
import { OverridesTable } from "./overrides-table";

export default function OverridePage(props: {
  params: Promise<{ namespaceId: string }>;
}) {
  const params = use(props.params);

  const { namespaceId } = params;

  const workspace = useWorkspaceNavigation();

  return (
    <div>
      <NamespaceNavbar
        activePage={{
          href: routes.ratelimits.overrides({ workspaceSlug: workspace.slug, namespaceId }),
          text: "Overrides",
        }}
        namespaceId={namespaceId}
      />
      <OverridesTable namespaceId={namespaceId} />
    </div>
  );
}
