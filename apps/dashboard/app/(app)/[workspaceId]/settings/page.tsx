"use client";
import { useWorkspace } from "@/providers/workspace-provider";
import { Loading } from "@unkey/ui";
import { useRouter } from "next/navigation";

export default function SettingsPage() {
  const { workspace, isLoading } = useWorkspace();
  const router = useRouter();

  if (workspace) {
    router.replace(`/${workspace.id}/settings/general`);
  }

  if (isLoading) {
    return (
      <div className="flex flex-col items-center justify-center h-screen w-full">
        <Loading size={18} />
      </div>
    );
  }

  if (!workspace) {
    router.push("/new");
  }
}
