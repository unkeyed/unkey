"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { Loading } from "@unkey/ui";
import { Suspense } from "react";
import { NamespaceNavbar } from "../namespace-navbar";
import { LogsClient } from "./components/logs-client";

export default function RatelimitLogsPage({
  params: { namespaceId },
}: {
  params: { namespaceId: string };
}) {
  const workspace = useWorkspaceNavigation();

  return (
    <div>
      <Suspense fallback={<Loading />}>
        <NamespaceNavbar
          namespaceId={namespaceId}
          activePage={{
            href: `/${workspace.slug}/ratelimits/${namespaceId}/logs`,
            text: "Logs",
          }}
        />
      </Suspense>
      <LogsClient namespaceId={namespaceId} />
    </div>
  );
}
