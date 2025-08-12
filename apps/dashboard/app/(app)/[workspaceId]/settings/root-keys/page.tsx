"use client";
import { useWorkspace } from "@/providers/workspace-provider";
import { redirect } from "next/navigation";
import { RootKeysListControlCloud } from "./components/control-cloud";
import { RootKeysListControls } from "./components/controls";
import { RootKeysList } from "./components/table/root-keys-list";
import { Navigation } from "./navigation";

export default function RootKeysPage() {
  const { workspace } = useWorkspace();

  if (!workspace) {
    redirect("/new");
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
        <RootKeysList workspaceId={workspace.id} />
      </div>
    </div>
  );
}
