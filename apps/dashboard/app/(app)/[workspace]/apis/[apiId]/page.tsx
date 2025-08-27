"use client";
import { LogsClient } from "@/app/(app)/[workspace]/apis/[apiId]/_overview/logs-client";
import { useWorkspace } from "@/providers/workspace-provider";
<<<<<<< HEAD
import { Loading } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useEffect } from "react";
=======
import { redirect, useRouter } from "next/navigation";
>>>>>>> eng-1894-a-user-can-access-settings-within-workspace-context
import { ApisNavbar } from "./api-id-navbar";

export default function ApiPage(props: { params: { apiId: string } }) {
  const apiId = props.params.apiId;
  const { workspace, error, isLoading } = useWorkspace();
  const router = useRouter();

<<<<<<< HEAD
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

    // If workspace exists, redirect to the correct workspace API page
    router.replace(`/${workspace.id}/apis/${apiId}`);
  }, [workspace, error, isLoading, router, apiId]);

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
=======
  if (workspace && !isLoading) {
    router.replace(`/${workspace?.slug}/apis/${apiId}`);
  }

  if (!workspace || error) {
    return redirect("/new");
>>>>>>> eng-1894-a-user-can-access-settings-within-workspace-context
  }

  return (
    <div className="min-h-screen">
      <ApisNavbar
        apiId={apiId}
        activePage={{
<<<<<<< HEAD
          href: `/${workspace.id}/apis/${apiId}`,
          text: "Requests",
        }}
        workspaceId={workspace.id}
      />
      <LogsClient apiId={apiId} workspaceId={workspace.id} />
=======
          href: `/${workspace?.slug}/apis/${apiId}`,
          text: "Requests",
        }}
        workspaceSlug={workspace?.slug ?? ""}
      />
      <LogsClient apiId={apiId} workspaceSlug={workspace?.slug ?? ""} />
>>>>>>> eng-1894-a-user-can-access-settings-within-workspace-context
    </div>
  );
}
