"use client";

import { useWorkspace } from "@/providers/workspace-provider";
import { Loading } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { ApisNavbar } from "../api-id-navbar";
import { SettingsClient } from "./components/settings-client";

type Props = {
  params: {
    apiId: string;
  };
};

export default function SettingsPage(props: Props) {
  const { apiId } = props.params;
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

    // If workspace exists, redirect to the correct workspace API settings page
    router.replace(`/${workspace.id}/apis/${apiId}/settings`);
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
    <div>
      <ApisNavbar
        apiId={apiId}
        activePage={{
          href: `/${workspace.id}/apis/${apiId}/settings`,
          text: "Settings",
        }}
        workspaceId={workspace.id}
      />
      <SettingsClient apiId={apiId} />
    </div>
  );
}
