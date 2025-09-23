"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { Loading } from "@unkey/ui";
import { Suspense } from "react";
import { LogsClient } from "./_overview/logs-client";
import { NamespaceNavbar } from "./namespace-navbar";

export default function RatelimitNamespacePage(props: {
  params: { namespaceId: string };
  searchParams: {
    identifier?: string;
  };
}) {
  const { namespaceId } = props.params;
  const workspace = useWorkspaceNavigation();

  return (
    <div>
      <Suspense fallback={<Loading type="spinner" />}>
        <NamespaceNavbar
          activePage={{
            href: `/${workspace.slug}/ratelimits/${namespaceId}`,
            text: "Requests",
          }}
          namespaceId={namespaceId}
        />
      </Suspense>
      <LogsClient namespaceId={namespaceId} />
    </div>
  );
}
