"use client";

import { Navbar } from "@/components/navigation/navbar";
import { useWorkspace } from "@/providers/workspace-provider";
import { Nodes } from "@unkey/icons";
import { Loading } from "@unkey/ui";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect } from "react";
import { ApiListClient } from "./_components/api-list-client";
import { CreateApiButton } from "./_components/create-api-button";

export default function ApisOverviewPage() {
  const { workspace, isLoading } = useWorkspace();
  const router = useRouter();
  const searchParams = useSearchParams();
  const isNewApi = searchParams?.get("new") === "true";

  useEffect(() => {
    // Return early while loading
    if (isLoading) {
      return;
    }

    // If no workspace, redirect to new workspace page
    if (!workspace) {
      router.replace("/new");
      return;
    }

    // If workspace exists, redirect to the correct workspace APIs page
    router.replace(`/${workspace.id}/apis`);
  }, [workspace, isLoading, router]);

  // Show loading state while workspace is loading
  if (isLoading) {
    return (
      <div className="flex flex-col items-center justify-center h-screen w-full">
        <Loading size={18} />
      </div>
    );
  }

  // Don't render anything if no workspace (will redirect)
  if (!workspace) {
    return null;
  }

  return (
    <div>
      <Navbar>
        <Navbar.Breadcrumbs icon={<Nodes />}>
          <Navbar.Breadcrumbs.Link href={`/${workspace.id}/apis`} active>
            APIs
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          <CreateApiButton key="createApi" defaultOpen={isNewApi} workspaceId={workspace.id} />
        </Navbar.Actions>
      </Navbar>
      <ApiListClient workspaceId={workspace.id} />
    </div>
  );
}
