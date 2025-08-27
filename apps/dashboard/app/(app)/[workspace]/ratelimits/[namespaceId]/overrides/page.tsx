"use client";

import { NamespaceNavbar } from "../namespace-navbar";
import { OverridesTable } from "./overrides-table";

export default function OverridePage({
  params: { workspaceId, namespaceId },
}: {
  params: { workspaceId: string; namespaceId: string };
}) {
  return (
    <div>
      <NamespaceNavbar
        activePage={{
          href: `/${workspaceId}/ratelimits/${namespaceId}/overrides`,
          text: "Overrides",
        }}
        namespaceId={namespaceId}
        workspaceId={workspaceId}
      />
      <OverridesTable namespaceId={namespaceId} />
    </div>
  );
}
