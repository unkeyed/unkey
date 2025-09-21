"use client";
import { useWorkspace } from "@/providers/workspace-provider";
import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { ApisNavbar } from "../../api-id-navbar";
import { KeysClient } from "./_components/keys-client";

export default function APIKeysPage(props: {
  params: {
    apiId: string;
    keyAuthId: string;
  };
}) {
  const apiId = props.params.apiId;
  const { workspace } = useWorkspace();
  const keyspaceId = props.params.keyAuthId;
  const router = useRouter();

  useEffect(() => {
    if (workspace) {
      router.replace(`/${workspace.slug}/apis/${apiId}/keys/${keyspaceId}`);
    }
  }, [workspace, router, apiId, keyspaceId]);
  return (
    <div>
      <ApisNavbar
        apiId={apiId}
        activePage={{
          href: `/${workspace?.slug}/apis/${apiId}/keys/${keyspaceId}`,
          text: "Keys",
        }}
      />
      <KeysClient apiId={apiId} keyspaceId={keyspaceId} />
    </div>
  );
}
