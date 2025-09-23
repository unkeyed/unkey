"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { Loading } from "@unkey/ui";
import { Suspense } from "react";
import { RootKeysListControlCloud } from "./components/control-cloud";
import { RootKeysListControls } from "./components/controls";
import { RootKeysList } from "./components/table/root-keys-list";
import { Navigation } from "./navigation";

export default function RootKeysPage() {
  const workspace = useWorkspaceNavigation();

  return (
    <div>
      <Suspense fallback={<Loading type="spinner" />}>
        <Navigation
          workspace={{
            id: workspace.id,
            name: workspace.name,
            slug: workspace.slug ?? "",
          }}
          activePage={{
            href: "root-keys",
            text: "Root Keys",
          }}
        />
      </Suspense>
      <div className="flex flex-col">
        <RootKeysListControls />
        <RootKeysListControlCloud />
        <RootKeysList />
      </div>
    </div>
  );
}
