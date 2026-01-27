"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { use } from "react";
import { ApisNavbar } from "../../api-id-navbar";
import { KeysClient } from "./_components/keys-client";

export default function APIKeysPage(props: {
  params: Promise<{
    apiId: string;
    keyAuthId: string;
  }>;
}) {
  const params = use(props.params);
  const apiId = params.apiId;
  const workspace = useWorkspaceNavigation();

  const keyspaceId = params.keyAuthId;

  return (
    <div>
      <ApisNavbar
        apiId={apiId}
        activePage={{
          href: `/${workspace.slug}/apis/${apiId}/keys/${keyspaceId}`,
          text: "Keys",
        }}
      />
      <KeysClient apiId={apiId} keyspaceId={keyspaceId} />
    </div>
  );
}
