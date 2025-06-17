"use client";

import { LogsClient } from "./_overview/logs-client";
import { NamespaceNavbar } from "./namespace-navbar";

export default function RatelimitNamespacePage(props: {
  params: { namespaceId: string };
  searchParams: {
    identifier?: string;
  };
}) {
  const namespaceId = props.params.namespaceId;
  return (
    <div>
      <NamespaceNavbar
        activePage={{
          href: `/ratelimits/${namespaceId}`,
          text: "Requests",
        }}
        namespaceId={namespaceId}
      />
      <LogsClient namespaceId={namespaceId} />
    </div>
  );
}
