"use client";
import { AppsList } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/_components/apps-list";
import { NoComputePlanBanner } from "@/app/(app)/[workspaceSlug]/projects/_components/no-compute-plan-banner";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { Plus } from "@unkey/icons";
import {
  Button,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import Link from "next/link";
import { useParams } from "next/navigation";

export default function ProjectAppsPage() {
  const params = useParams();
  const workspace = useWorkspaceNavigation();
  const projectId = typeof params?.projectId === "string" ? params.projectId : "";

  return (
    <PageContainer className="flex-1">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Apps</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <Button
            size="md"
            variant="primary"
            render={
              <Link href={routes.projects.apps.new({ workspaceSlug: workspace.slug, projectId })} />
            }
          >
            <Plus iconSize="sm-regular" />
            Create app
          </Button>
        </PageHeaderActions>
      </PageHeader>
      <PageBody className="flex-1">
        <NoComputePlanBanner />
        <AppsList />
      </PageBody>
    </PageContainer>
  );
}
