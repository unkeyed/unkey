"use client";

import { LogsClient } from "./_overview/logs-client";
import { NamespaceNavbar } from "./namespace-navbar";

export default function RatelimitNamespacePage(props: {
  params: { workspaceId: string; namespaceId: string };
  searchParams: {
    identifier?: string;
  };
}) {
  const { workspaceId } = props.params;
  const namespaceId = props.params.namespaceId;
  return (
    <div>
      <NamespaceNavbar
        activePage={{
          href: `/${workspaceId}/ratelimits/${namespaceId}`,
          text: "Requests",
        }}
        namespaceId={namespaceId}
        workspaceId={workspaceId} 
      />
      <LogsClient namespaceId={namespaceId} />
    </div>
  );
}
