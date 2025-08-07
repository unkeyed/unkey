"use client";

import { ApisNavbar } from "../../../api-id-navbar";
import { KeyDetailsLogsClient } from "./logs-client";

export default function KeyDetailsPage(props: {
  params: { apiId: string; keyAuthId: string; keyId: string; workspaceId: string };
}) {
  const { apiId, keyAuthId: keyspaceId, keyId, workspaceId } = props.params;

  return (
    <div className="w-full">
      <ApisNavbar
        apiId={apiId}
        keyspaceId={keyspaceId}
        keyId={keyId}
        activePage={{
          href: `/${workspaceId}/apis/${apiId}/keys/${keyspaceId}`,
          text: "Keys",
        }}
        workspaceId={workspaceId}
      />
      <KeyDetailsLogsClient apiId={apiId} keyspaceId={keyspaceId} keyId={keyId} />
    </div>
  );
}
