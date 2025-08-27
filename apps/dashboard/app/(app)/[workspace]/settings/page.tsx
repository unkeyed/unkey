"use client";
import { useWorkspace } from "@/providers/workspace-provider";
import { Loading } from "@unkey/ui";
import { useParams, useRouter } from "next/navigation";
import { useEffect } from "react";

export default function SettingsPage() {
  const { workspace, isLoading } = useWorkspace();
  const router = useRouter();
  const params = useParams();
  const workspaceId = params?.workspaceId as string;

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

    // If current workspace ID matches the URL workspace ID, redirect to general settings
    if (workspace.id === workspaceId) {
      router.replace(`/${workspace.slug}/settings/general`);
      return;
    }

    // If workspace IDs don't match, redirect to the correct workspace
    router.replace(`/${workspace.slug}/settings`);
  }, [workspace, isLoading, workspaceId, router]);

  // Show loading state while redirecting
  if (isLoading) {
    return (
      <output
        className="flex flex-col items-center justify-center h-screen w-full"
        aria-busy="true"
        aria-live="polite"
      >
        <Loading size={18} />
      </output>
    );
  }

  // Return null to avoid render-side effects and double navigation
  return null;
}
