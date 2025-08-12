"use client";

import { useWorkspace } from "@/providers/workspace-provider";
import { Loading } from "@unkey/ui";
import { redirect, useRouter } from "next/navigation";
import { ApisNavbar } from "../../../api-id-navbar";
import { KeyDetailsLogsClient } from "./logs-client";

export default function KeyDetailsPage(props: {
  params: { apiId: string; keyAuthId: string; keyId: string };
}) {
  const { apiId, keyAuthId: keyspaceId, keyId } = props.params;
  const { workspace, isLoading, error } = useWorkspace();
  const router = useRouter();

  if (isLoading) {
    return <Loading size={18} />;
  }

  if (!workspace || error) {
    return redirect("/new");
  }

  router.replace(`/${workspace?.id}/apis/${apiId}/keys/${keyspaceId}`);
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
