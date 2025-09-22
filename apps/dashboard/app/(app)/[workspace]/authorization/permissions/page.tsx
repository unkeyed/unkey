"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { PermissionsListControlCloud } from "./components/control-cloud";
import { PermissionListControls } from "./components/controls";
import { PermissionsList } from "./components/table/permissions-list";
import { Navigation } from "./navigation";

export default function PermissionsPage() {
  const workspace = useWorkspaceNavigation();

  return (
    workspace && (
      <div>
        <Navigation />
        <div className="flex flex-col">
          <PermissionListControls />
          <PermissionsListControlCloud />
          <PermissionsList />
        </div>
      </div>
    )
  );
}
