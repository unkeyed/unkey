"use client";

import { NamespaceNavbar } from "../namespace-navbar";
import { LogsClient } from "./components/logs-client";

export default function RatelimitLogsPage({
  params: { namespaceId },
}: {
  params: { namespaceId: string };
}) {
  return (
    <div>
      <NamespaceNavbar
        namespaceId={namespaceId}
        activePage={{
          href: `/ratelimits/${namespaceId}/logs`,
          text: "Logs",
        }}
      />
      <LogsClient namespaceId={namespaceId} />
    </div>
  );
}
