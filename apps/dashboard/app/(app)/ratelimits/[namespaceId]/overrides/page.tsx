"use client";

import { NamespaceNavbar } from "../namespace-navbar";
import { OverridesTable } from "./overrides-table";

export default function OverridePage({
  params: { namespaceId },
}: {
  params: { namespaceId: string };
}) {
  return (
    <div>
      <NamespaceNavbar
        activePage={{
          href: `/ratelimits/${namespaceId}/overrides`,
          text: "Overrides",
        }}
        namespaceId={namespaceId}
      />
      <OverridesTable namespaceId={namespaceId} />
    </div>
  );
}
