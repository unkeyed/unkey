"use client";
import { useWorkspace } from "@/providers/workspace-provider";
import { RolesListControlCloud } from "./components/control-cloud";
import { RoleListControls } from "./components/controls";
import { RolesList } from "./components/table/roles-list";
import { Navigation } from "./navigation";

export default function RolesPage() {
  const { workspace } = useWorkspace();

  return (
    workspace && (
      <div>
        <Navigation />
        <div className="flex flex-col">
          <RoleListControls />
          <RolesListControlCloud />
          <RolesList />
        </div>
      </div>
    )
  );
}
