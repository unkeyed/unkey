"use client";

import { ApisNavbar } from "../../../api-id-navbar";
import { KeyDetailsLogsClient } from "./logs-client";

export default function KeyDetailsPage(props: {
  params: { apiId: string; keyAuthId: string; keyId: string };
}) {
  const { apiId, keyAuthId: keyspaceId, keyId } = props.params;

  return (
    <div className="w-full">
      <ApisNavbar
        apiId={apiId}
        keyspaceId={keyspaceId}
        keyId={keyId}
        activePage={{
          href: `/apis/${apiId}/keys/${keyspaceId}`,
          text: "Keys",
        }}
      />
      <KeyDetailsLogsClient apiId={apiId} keyspaceId={keyspaceId} keyId={keyId} />
    </div>
  );
}
