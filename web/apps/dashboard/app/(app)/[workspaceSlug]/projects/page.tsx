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
import { CreateProjectButton } from "./_components/create-project-button";
import { useDeployGate } from "./_components/hooks/use-deploy-gate";
import { ProjectsList } from "./_components/list";
import { EmptyProjects } from "./_components/list/empty-projects";

export default function ProjectsPage() {
  const workspace = useWorkspaceNavigation();
  const searchParams = useSearchParams();
  const isNewProject = searchParams.get("new") === "true";
  const projects = useLiveQuery((q) => q.from({ project: collection.projects }));

  // Hook order: must run unconditionally, before the empty-state early return.
  const { gated } = useDeployGate();

  // With no projects, a gated workspace falls through to the list, which shows
  // the "Choose a plan" paywall as its empty state. Everyone else gets the
  // normal create-your-first-project screen.
  if (!projects.isLoading && projects.data.length === 0 && !gated) {
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
        <ProjectsList />
        <NewNavigationBanner />
      </PageBody>
    </PageContainer>
  );
}
