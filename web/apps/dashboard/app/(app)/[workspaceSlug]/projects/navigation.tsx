"use client";
import { Navbar } from "@/components/navigation/navbar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { Cube } from "@unkey/icons";
import { useSearchParams } from "next/navigation";
import { CreateProjectButton } from "./_components/create-project-button";

export function ProjectsListNavigation() {
  const workspace = useWorkspaceNavigation();
  const searchParams = useSearchParams();
  const isNewProject = searchParams.get("new") === "true";
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Cube iconSize="md-medium" className="text-gray-12" />}>
        <Navbar.Breadcrumbs.Link
          href={routes.projects.list({ workspaceSlug: workspace.slug })}
          active
        >
          Projects
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
      <Navbar.Actions>
        <CreateProjectButton defaultOpen={isNewProject} workspaceSlug={workspace.slug} />
      </Navbar.Actions>
    </Navbar>
  );
}
