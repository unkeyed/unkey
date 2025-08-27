"use client";

import { useWorkspace } from "@/providers/workspace-provider";
import { redirect, useRouter } from "next/navigation";
import { LogsClient } from "./_overview/logs-client";
import { NamespaceNavbar } from "./namespace-navbar";

export default function RatelimitNamespacePage(props: {
  params: { workspaceId: string; namespaceId: string };
  searchParams: {
    identifier?: string;
  };
}) {
  const { workspaceId, namespaceId } = props.params;
  const router = useRouter();
  const { workspace } = useWorkspace();

  if (!workspace) {
    return redirect("/new");
  }

  if (workspaceId !== workspace.id) {
    router.replace(`/${workspace.id}/ratelimits/${namespaceId}`);
  }
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
