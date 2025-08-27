"use client";
import { LogsClient } from "@/app/(app)/[workspace]/apis/[apiId]/_overview/logs-client";
import { useWorkspace } from "@/providers/workspace-provider";
import { Loading } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { ApisNavbar } from "./api-id-navbar";

export default function ApiPage(props: { params: { apiId: string } }) {
  const apiId = props.params.apiId;
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
  }

  return (
    <div className="min-h-screen">
      <ApisNavbar
        apiId={apiId}
        activePage={{
          href: `/${workspace.id}/apis/${apiId}`,
          text: "Requests",
        }}
        workspaceId={workspace.id}
      />
      <LogsClient apiId={apiId} workspaceId={workspace.id} />
    </div>
  );
}
