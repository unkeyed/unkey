"use client";
import { DeployFeedbackButton } from "@/components/dashboard/deploy-feedback-button";
import { Navbar } from "@/components/navigation/navbar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
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
        <Navbar.Breadcrumbs.Link href={`/${workspace.slug}/projects`} active>
          Projects
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
      <Navbar.Actions>
        <CreateProjectButton defaultOpen={isNewProject} workspaceSlug={workspace.slug} />
        <DeployFeedbackButton />
      </Navbar.Actions>
    </Navbar>
  );
}
