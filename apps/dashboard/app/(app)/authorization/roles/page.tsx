"use client";
import { RolesList } from "./components/table/roles-list";
import { Navigation } from "./navigation";

export default function RolesPage() {
  return (
    <div>
      <Navigation />
      <div className="flex flex-col">
        <RolesList />
      </div>
    </div>
  );
}
