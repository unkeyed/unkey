"use client";
import { useWorkspace } from "@/providers/workspace-provider";
import { Loading } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { RootKeysListControlCloud } from "./components/control-cloud";
import { RootKeysListControls } from "./components/controls";
import { RootKeysList } from "./components/table/root-keys-list";
import { Navigation } from "./navigation";

export default function RootKeysPage() {
  const { workspace, isLoading } = useWorkspace();
  const router = useRouter();

  useEffect(() => {
    // Only redirect when workspace is explicitly null (not undefined during loading)
    if (!isLoading && workspace === null) {
      router.replace("/new");
    }
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
      <Navigation
        workspace={{
          id: workspace.id,
          name: workspace.name,
        }}
        activePage={{
          href: "root-keys",
          text: "Root Keys",
        }}
      />
      <div className="flex flex-col">
        <RootKeysListControls />
        <RootKeysListControlCloud />
        <RootKeysList />
      </div>
    </div>
  );
}
