"use client";
import { Navbar } from "@/components/navigation/navbar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { Cube } from "@unkey/icons";
import { CreateProjectDialog } from "./_components/create-project/create-project-dialog";

export function ProjectsListNavigation() {
  const workspace = useWorkspaceNavigation();
  return (
    <Navbar>
      <Navbar.Breadcrumbs
        icon={<Cube iconSize="md-medium" className="text-gray-12" />}
      >
        <Navbar.Breadcrumbs.Link href={`/${workspace.slug}/projects`} active>
          Projects
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
      <CreateProjectDialog />
    </Navbar>
  );
}
