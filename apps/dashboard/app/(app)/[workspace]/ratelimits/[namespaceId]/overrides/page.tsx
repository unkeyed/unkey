"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { Loading } from "@unkey/ui";
import { Suspense } from "react";
import { NamespaceNavbar } from "../namespace-navbar";
import { OverridesTable } from "./overrides-table";

export default function OverridePage({
  params: { namespaceId },
}: {
  params: { namespaceId: string };
}) {
  const workspace = useWorkspaceNavigation();

  return (
    <div>
      <Suspense fallback={<Loading />}>
        <NamespaceNavbar
          activePage={{
            href: `/${workspace.slug}/ratelimits/${namespaceId}/overrides`,
            text: "Overrides",
          }}
          namespaceId={namespaceId}
        />
      </Suspense>
      <OverridesTable namespaceId={namespaceId} />
    </div>
  );
}
