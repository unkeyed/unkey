"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { Loading } from "@unkey/ui";
import { Suspense } from "react";
import { ApisNavbar } from "../../api-id-navbar";
import { KeysClient } from "./_components/keys-client";

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
      <Suspense fallback={<Loading type="spinner" />}>
        <ApisNavbar
          apiId={apiId}
          activePage={{
            href: `/${workspace.slug}/apis/${apiId}/keys/${keyspaceId}`,
            text: "Keys",
          }}
        />
        <KeysClient apiId={apiId} keyspaceId={keyspaceId} />
      </Suspense>
    </div>
  );
}
