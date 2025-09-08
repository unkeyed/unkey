"use client";
import { useWorkspace } from "@/providers/workspace-provider";
import { useRouter } from "next/navigation";
import { PermissionsListControlCloud } from "./components/control-cloud";
import { PermissionListControls } from "./components/controls";
import { PermissionsList } from "./components/table/permissions-list";
import { Navigation } from "./navigation";

export default function PermissionsPage() {
  const router = useRouter();
  const { workspace } = useWorkspace();

  if (!workspace) {
    router.push("/new");
  }

  return (
    <div>
      <Navigation />
      <div className="flex flex-col">
        <PermissionListControls />
        <PermissionsListControlCloud />
        <PermissionsList />
      </div>
    </div>
  );
}
