"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { RolesListControlCloud } from "./components/control-cloud";
import { RoleListControls } from "./components/controls";
import { RolesList } from "./components/table/roles-list";
import { Navigation } from "./navigation";

export default function RolesPage() {
  const workspace = useWorkspaceNavigation();

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
