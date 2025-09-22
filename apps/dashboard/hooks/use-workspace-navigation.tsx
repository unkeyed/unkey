import { redirect } from "next/navigation";
import { useWorkspace } from "@/providers/workspace-provider";

export const useWorkspaceNavigation = () => {
  const { workspace, isLoading: isWorkspaceLoading } = useWorkspace();

  if (isWorkspaceLoading) {
    throw new Promise(() => {});
  }

  if (!workspace) {
    redirect("/new");
    throw new Error("No workspace");
  }

  return workspace;
};
