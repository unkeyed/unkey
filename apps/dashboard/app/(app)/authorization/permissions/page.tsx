"use client";
import { PermissionsList } from "./components/table/permissions-list";
import { Navigation } from "./navigation";

export default function RolesPage() {
  return (
    <div>
      <Navigation />
      <div className="flex flex-col">
        {/*   <RoleListControls /> */}
        {/*   <RolesListControlCloud /> */}
        <PermissionsList />
      </div>
    </div>
  );
}
