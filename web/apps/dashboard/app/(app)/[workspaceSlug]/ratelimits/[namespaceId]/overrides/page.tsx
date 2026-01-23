"use client";;
import { use } from "react";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { NamespaceNavbar } from "../namespace-navbar";
import { OverridesTable } from "./overrides-table";

export default function OverridePage(
  props: {
    params: Promise<{ namespaceId: string }>;
  }
) {
  const params = use(props.params);

  const {
    namespaceId
  } = params;

  const workspace = useWorkspaceNavigation();

  return (
    <div>
      <NamespaceNavbar
        activePage={{
          href: `/${workspace.slug}/ratelimits/${namespaceId}/overrides`,
          text: "Overrides",
        }}
        namespaceId={namespaceId}
      />
      <OverridesTable namespaceId={namespaceId} />
    </div>
  );
}
