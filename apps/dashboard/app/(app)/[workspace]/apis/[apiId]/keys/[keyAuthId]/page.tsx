"use client";
import { useWorkspace } from "@/providers/workspace-provider";
<<<<<<< HEAD
import { Loading } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useEffect } from "react";
=======
import { redirect, useRouter } from "next/navigation";
>>>>>>> eng-1894-a-user-can-access-settings-within-workspace-context
import { ApisNavbar } from "../../api-id-navbar";
import { KeysClient } from "./_components/keys-client";

export default function APIKeysPage(props: {
  params: {
    apiId: string;
    keyAuthId: string;
  };
}) {
  const apiId = props.params.apiId;
<<<<<<< HEAD
  const { workspace, error, isLoading } = useWorkspace();
  const keyspaceId = props.params.keyAuthId;
  const router = useRouter();

  useEffect(() => {
    // Return early while loading
    if (isLoading) {
      return;
    }

    // If no workspace or error, redirect to new workspace page
    if (!workspace || error) {
      router.replace("/new");
      return;
    }

    // If workspace exists, redirect to the correct workspace keys page
    router.replace(`/${workspace.id}/apis/${apiId}/keys/${keyspaceId}`);
  }, [workspace, error, isLoading, router, apiId, keyspaceId]);

  // Show loading state while workspace is loading
  if (isLoading) {
    return (
      <div className="flex flex-col items-center justify-center h-screen w-full">
        <Loading size={18} />
      </div>
    );
  }

  // Don't render anything if no workspace or error (will redirect)
  if (!workspace || error) {
    return null;
  }

=======
  const { workspace, error } = useWorkspace();
  const keyspaceId = props.params.keyAuthId;
  const router = useRouter();

  if (!workspace || error) {
    return redirect("/new");
  }

  router.replace(`/${workspace?.slug}/apis/${apiId}/keys/${keyspaceId}`);
>>>>>>> eng-1894-a-user-can-access-settings-within-workspace-context
  return (
    <div>
      <ApisNavbar
        activePage={{
<<<<<<< HEAD
          href: `/${workspace.id}/apis/${apiId}/keys/${keyspaceId}`,
          text: "Keys",
        }}
        apiId={apiId}
        workspaceId={workspace.id}
      />
      <KeysClient apiId={props.params.apiId} keyspaceId={keyspaceId} workspaceId={workspace.id} />
=======
          href: `/${workspace?.slug}/apis/${apiId}/keys/${keyspaceId}`,
          text: "Keys",
        }}
        apiId={apiId}
        workspaceSlug={workspace?.slug ?? ""}
      />
      <KeysClient
        apiId={props.params.apiId}
        keyspaceId={keyspaceId}
        workspaceSlug={workspace?.slug ?? ""}
      />
>>>>>>> eng-1894-a-user-can-access-settings-within-workspace-context
    </div>
  );
}
