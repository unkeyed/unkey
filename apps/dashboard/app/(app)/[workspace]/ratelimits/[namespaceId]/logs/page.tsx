"use client";

import { useWorkspace } from "@/providers/workspace-provider";
import { NamespaceNavbar } from "../namespace-navbar";
import { LogsClient } from "./components/logs-client";

export default function RatelimitLogsPage({
  params: { namespaceId },
}: {
  params: { namespaceId: string };
}) {
  const { workspace } = useWorkspace();
  return (
    <div>
      <NamespaceNavbar
        namespaceId={namespaceId}
        activePage={{
          href: `/${workspace?.slug}/ratelimits/${namespaceId}/logs`,
          text: "Logs",
        }}
      />
      <LogsClient namespaceId={namespaceId} />
    </div>
  );
}
