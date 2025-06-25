"use client";
import { ApisNavbar } from "../../api-id-navbar";
import { KeysClient } from "./_components/keys-client";

export default function APIKeysPage(props: {
  params: {
    apiId: string;
    keyAuthId: string;
  };
}) {
  const apiId = props.params.apiId;
  const keyspaceId = props.params.keyAuthId;

  return (
    <div>
      <ApisNavbar
        activePage={{
          href: `/apis/${apiId}/keys/${keyspaceId}`,
          text: "Keys",
        }}
        apiId={apiId}
      />
      <KeysClient apiId={props.params.apiId} keyspaceId={keyspaceId} />
    </div>
  );
}
