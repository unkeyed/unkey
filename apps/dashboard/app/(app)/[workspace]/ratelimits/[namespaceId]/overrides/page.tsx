"use client";

import { useWorkspace } from "@/providers/workspace-provider";
import { NamespaceNavbar } from "../namespace-navbar";
import { OverridesTable } from "./overrides-table";

export default function OverridePage({
  params: { namespaceId },
}: {
  params: { namespaceId: string };
}) {
  const { workspace } = useWorkspace();
  return (
    <div>
      <NamespaceNavbar
        activePage={{
          href: `/${workspace?.slug}/ratelimits/${namespaceId}/overrides`,
          text: "Overrides",
        }}
        namespaceId={namespaceId}
      />
      <OverridesTable namespaceId={namespaceId} />
    </div>
  );
}
