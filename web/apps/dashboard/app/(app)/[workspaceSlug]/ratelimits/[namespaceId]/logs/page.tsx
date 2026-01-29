"use client";
import { use } from "react";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";

import { NamespaceNavbar } from "../namespace-navbar";
import { LogsClient } from "./components/logs-client";

export default function RatelimitLogsPage(props: {
  params: Promise<{ namespaceId: string }>;
}) {
  const params = use(props.params);

  const { namespaceId } = params;

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
