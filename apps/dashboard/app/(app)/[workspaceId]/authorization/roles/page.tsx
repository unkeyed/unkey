"use client";
import { RolesListControlCloud } from "./components/control-cloud";
import { RoleListControls } from "./components/controls";
import { RolesList } from "./components/table/roles-list";
import { Navigation } from "./navigation";

export default function RolesPage({ params }: { params: { workspaceId: string } }) {
  return (
    <div>
      <Navigation workspaceId={params.workspaceId} />
      <div className="flex flex-col">
        <RoleListControls />
        <RolesListControlCloud />
        <RolesList />
      </div>
    </div>
  );
}
