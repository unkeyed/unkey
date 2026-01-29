"use client";
import { use } from "react";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { LogsClient } from "./_overview/logs-client";
import { NamespaceNavbar } from "./namespace-navbar";

export default function RatelimitNamespacePage(props: {
  params: Promise<{ namespaceId: string }>;
  searchParams: Promise<{
    identifier?: string;
  }>;
}) {
  const params = use(props.params);
  const { namespaceId } = params;
  const workspace = useWorkspaceNavigation();

  return (
    <div>
      <NamespaceNavbar
        activePage={{
          href: `/${workspace.slug}/ratelimits/${namespaceId}`,
          text: "Requests",
        }}
        namespaceId={namespaceId}
      />
      <LogsClient namespaceId={namespaceId} />
    </div>
  );
}
