"use client";
import { useWorkspace } from "@/providers/workspace-provider";
import { Loading } from "@unkey/ui";
import { redirect, useRouter } from "next/navigation";
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
  const router = useRouter();

  if (isLoading) {
    return (
      <div className="flex flex-col items-center justify-center h-screen w-full">
        <Loading size={18} />
      </div>
    );
  }

  if (!workspace || error) {
    return redirect("/new");
  }

  router.replace(`/${workspace?.id}/apis/${apiId}/keys/${keyspaceId}`);
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
