"use client";
import { useWorkspaceWithRedirect } from "@/hooks/use-workspace-with-redirect";
import { useRouter } from "next/navigation";
import { ApisNavbar } from "../../api-id-navbar";
import { KeysClient } from "./_components/keys-client";

export default function APIKeysPage(props: {
  params: {
    apiId: string;
    keyAuthId: string;
  };
}) {
  const apiId = props.params.apiId;
  const { workspace } = useWorkspaceWithRedirect();
  const keyspaceId = props.params.keyAuthId;
  const router = useRouter();

  router.replace(`/${workspace.slug}/apis/${apiId}/keys/${keyspaceId}`);
  return (
    <div>
      <ApisNavbar
        activePage={{
          href: `/${workspace.slug}/apis/${apiId}/keys/${keyspaceId}`,
          text: "Keys",
        }}
        apiId={apiId}
      />
      <KeysClient apiId={props.params.apiId} keyspaceId={keyspaceId} />
    </div>
  );
}
