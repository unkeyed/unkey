"use client";
import { RolesList } from "./components/table/keys-list";
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
