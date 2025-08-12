"use client";

import { useWorkspace } from "@/providers/workspace-provider";
import { redirect } from "next/navigation";
import { ApisNavbar } from "../../../api-id-navbar";
import { KeyDetailsLogsClient } from "./logs-client";

export default function KeyDetailsPage(props: {
  params: { apiId: string; keyAuthId: string; keyId: string };
}) {
  const { apiId, keyAuthId: keyspaceId, keyId } = props.params;
  const { workspace, isLoading, error } = useWorkspace();

  if (!workspace && !isLoading && error) {
    return redirect("/new");
  }

  return (
    <div className="w-full">
      <ApisNavbar
        apiId={apiId}
        keyspaceId={keyspaceId}
        keyId={keyId}
        activePage={{
          href: `/${workspace?.id}/apis/${apiId}/keys/${keyspaceId}`,
          text: "Keys",
        }}
        workspaceId={workspace?.id ?? ""}
      />
      <KeyDetailsLogsClient apiId={apiId} keyspaceId={keyspaceId} keyId={keyId} />
    </div>
  );
}
