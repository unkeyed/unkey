"use client";

import { useWorkspace } from "@/providers/workspace-provider";
<<<<<<< HEAD
import { Loading } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useEffect } from "react";
=======
import { redirect, useRouter } from "next/navigation";
>>>>>>> eng-1894-a-user-can-access-settings-within-workspace-context
import { ApisNavbar } from "../../../api-id-navbar";
import { KeyDetailsLogsClient } from "./logs-client";

export default function KeyDetailsPage(props: {
  params: { apiId: string; keyAuthId: string; keyId: string };
}) {
  const { apiId, keyAuthId: keyspaceId, keyId } = props.params;
<<<<<<< HEAD
  const { workspace, error, isLoading } = useWorkspace();
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

    // If workspace exists, redirect to the correct workspace key details page
    router.replace(`/${workspace.id}/apis/${apiId}/keys/${keyspaceId}/${keyId}`);
  }, [workspace, error, isLoading, router, apiId, keyspaceId, keyId]);

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
  const router = useRouter();

  if (!workspace || error) {
    return redirect("/new");
  }

  router.replace(`/${workspace?.slug}/apis/${apiId}/keys/${keyspaceId}/${keyId}`);
>>>>>>> eng-1894-a-user-can-access-settings-within-workspace-context
  return (
    <div className="w-full">
      <ApisNavbar
        apiId={apiId}
        keyspaceId={keyspaceId}
        keyId={keyId}
        activePage={{
<<<<<<< HEAD
          href: `/${workspace.id}/apis/${apiId}/keys/${keyspaceId}/${keyId}`,
          text: "Keys",
        }}
        workspaceId={workspace.id}
=======
          href: `/${workspace?.slug}/apis/${apiId}/keys/${keyspaceId}/${keyId}`,
          text: "Keys",
        }}
        workspaceSlug={workspace?.slug ?? ""}
>>>>>>> eng-1894-a-user-can-access-settings-within-workspace-context
      />
      <KeyDetailsLogsClient apiId={apiId} keyspaceId={keyspaceId} keyId={keyId} />
    </div>
  );
}
