"use client";

import { NamespaceNavbar } from "../namespace-navbar";
import { LogsClient } from "./components/logs-client";

export default function RatelimitLogsPage({
  params: { workspaceId, namespaceId },
}: {
  params: { workspaceId: string; namespaceId: string };
}) {
  return (
    <div>
      <NamespaceNavbar
        namespaceId={namespaceId}
        activePage={{
          href: `/ratelimits/${namespaceId}/logs`,
          text: "Logs",
        }}
        workspaceId={workspaceId}
      />
      <LogsClient namespaceId={namespaceId} />
    </div>
  );
}
