"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
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
          href: routes.apis.keys.list({
            workspaceSlug: workspace.slug,
            apiId,
            keyAuthId: keyspaceId,
          }),
          text: "Keys",
        }}
      />
      <KeysClient apiId={apiId} keyspaceId={keyspaceId} />
    </div>
  );
}
