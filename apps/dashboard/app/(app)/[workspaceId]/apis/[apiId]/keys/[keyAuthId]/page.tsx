"use client";
import { ApisNavbar } from "../../api-id-navbar";
import { KeysClient } from "./_components/keys-client";

export default function APIKeysPage(props: {
  params: {
    workspaceId: string;
    apiId: string;
    keyAuthId: string;
  };
}) {
  const apiId = props.params.apiId;
  const workspaceId = props.params.workspaceId;
  const keyspaceId = props.params.keyAuthId;
  return (
    <div>
      <ApisNavbar
        activePage={{
          href: `/${workspaceId}/apis/${apiId}/keys/${keyspaceId}`,
          text: "Keys",
        }}
        apiId={apiId}
        workspaceId={workspaceId}
      />
      <KeysClient apiId={props.params.apiId} keyspaceId={keyspaceId} workspaceId={workspaceId} />
    </div>
  );
}
