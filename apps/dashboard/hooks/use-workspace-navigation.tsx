import { useWorkspace } from "@/providers/workspace-provider";
import { redirect } from "next/navigation";

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
