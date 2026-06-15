"use client";

import { NewNavigationBanner } from "@/components/navigation/new-navigation-banner";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { useLiveQuery } from "@tanstack/react-db";
import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import { useSearchParams } from "next/navigation";
import { ProjectsListControls } from "./_components/controls";
import { CreateProjectButton } from "./_components/create-project-button";
import { ProjectsList } from "./_components/list";
import { EmptyProjects } from "./_components/list/empty-projects";

export default function ProjectsPage() {
  const workspace = useWorkspaceNavigation();
  const searchParams = useSearchParams();
  const isNewProject = searchParams.get("new") === "true";
  const projects = useLiveQuery((q) => q.from({ project: collection.projects }));

  if (!projects.isLoading && projects.data.length === 0) {
    return <EmptyProjects />;
  }

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Projects</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <CreateProjectButton defaultOpen={isNewProject} workspaceSlug={workspace.slug} />
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        <ProjectsListControls />
        <ProjectsList />
        <NewNavigationBanner />
      </PageBody>
    </PageContainer>
  );
}
