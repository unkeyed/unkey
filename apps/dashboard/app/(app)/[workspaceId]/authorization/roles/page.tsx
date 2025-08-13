"use client";
import { useWorkspace } from "@/providers/workspace-provider";
import { useRouter } from "next/navigation";
import { RolesListControlCloud } from "./components/control-cloud";
import { RoleListControls } from "./components/controls";
import { RolesList } from "./components/table/roles-list";
import { Navigation } from "./navigation";

export default function RolesPage() {
  const router = useRouter();
  const { workspace } = useWorkspace();

  if (!workspace) {
    router.push("/new");
  }

  return (
    <div>
      <Navigation workspaceId={workspace?.id ?? ""} />
      <div className="flex flex-col">
        <RoleListControls />
        <RolesListControlCloud />
        <RolesList />
      </div>
    </div>
  );
}
