"use client";
import { useWorkspace } from "@/providers/workspace-provider";
import { redirect } from "next/navigation";
import { ApisNavbar } from "../../api-id-navbar";
import { KeysClient } from "./_components/keys-client";

export default function APIKeysPage(props: {
  params: {
    apiId: string;
    keyAuthId: string;
  };
}) {
  const apiId = props.params.apiId;
  const { workspace, isLoading, error } = useWorkspace();
  const keyspaceId = props.params.keyAuthId;
  if (!workspace && !isLoading && error) {
    return redirect("/new");
  }
  return (
    <div>
      <ApisNavbar
        activePage={{
          href: `/${workspace?.id}/apis/${apiId}/keys/${keyspaceId}`,
          text: "Keys",
        }}
        apiId={apiId}
        workspaceId={workspace?.id ?? ""}
      />
      <KeysClient
        apiId={props.params.apiId}
        keyspaceId={keyspaceId}
        workspaceId={workspace?.id ?? ""}
      />
    </div>
  );
}
