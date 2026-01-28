"use client";
import { use } from "react";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { ApisNavbar } from "../../../api-id-navbar";
import { KeyDetailsLogsClient } from "./logs-client";
export default function KeyDetailsPage(props: {
  params: Promise<{ apiId: string; keyAuthId: string; keyId: string }>;
}) {
  const params = use(props.params);
  const { apiId, keyAuthId: keyspaceId, keyId } = params;
  const workspace = useWorkspaceNavigation();

  return (
    <div className="w-full">
      <ApisNavbar
        apiId={apiId}
        keyspaceId={keyspaceId}
        keyId={keyId}
        activePage={{
          href: `/${workspace.slug}/apis/${apiId}/keys/${keyspaceId}/${keyId}`,
          text: "Keys",
        }}
      />
      <KeyDetailsLogsClient apiId={apiId} keyspaceId={keyspaceId} keyId={keyId} />
    </div>
  );
}
