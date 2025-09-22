"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { NamespaceNavbar } from "../namespace-navbar";
import { OverridesTable } from "./overrides-table";
import { redirect } from "next/navigation";
import { Loading } from "@unkey/ui";

export default function OverridePage({
  params: { namespaceId },
}: {
  params: { namespaceId: string };
}) {
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
