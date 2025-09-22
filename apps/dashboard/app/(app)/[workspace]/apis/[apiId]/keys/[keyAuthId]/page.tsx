"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { redirect } from "next/navigation";
import { ApisNavbar } from "../../api-id-navbar";
import { KeysClient } from "./_components/keys-client";
import { Loading } from "@unkey/ui";

export default function APIKeysPage(props: {
  params: {
    apiId: string;
    keyAuthId: string;
  };
}) {
  const apiId = props.params.apiId;
  const workspace = useWorkspaceNavigation();

  const keyspaceId = props.params.keyAuthId;

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
