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
import { DeployGateDebugBar } from "./_components/deploy-gate-debug-bar";
import { ProjectsList } from "./_components/list";
import { EmptyProjects } from "./_components/list/empty-projects";

export default function ProjectsPage() {
  const workspace = useWorkspaceNavigation();
  const searchParams = useSearchParams();
  const isNewProject = searchParams.get("new") === "true";
  const projects = useLiveQuery((q) => q.from({ project: collection.projects }));

  const isEmpty = !projects.isLoading && projects.data.length === 0;

  return (
    <>
      {isEmpty ? (
        <EmptyProjects />
      ) : (
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
      )}
      <DeployGateDebugBar />
    </>
  );
}
