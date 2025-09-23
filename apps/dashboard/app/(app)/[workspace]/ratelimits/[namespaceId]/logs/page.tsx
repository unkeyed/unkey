"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";

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
      <NamespaceNavbar
        namespaceId={namespaceId}
        activePage={{
          href: `/${workspace.slug}/ratelimits/${namespaceId}/logs`,
          text: "Logs",
        }}
      />
      <LogsClient namespaceId={namespaceId} />
    </div>
  );
}
