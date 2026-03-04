"use client";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { Plus } from "@unkey/icons";
import Link from "next/link";

export const CreateProjectDialog = () => {
  const workspace = useWorkspaceNavigation();

  return (
    <Link href={`/${workspace.slug}/projects/new`}>
      <NavbarActionButton title="Create new project" className="cursor-pointer">
        <Plus />
        Create new project
      </NavbarActionButton>
    </Link>
  );
};
