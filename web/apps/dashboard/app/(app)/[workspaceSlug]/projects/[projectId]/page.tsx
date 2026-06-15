"use client";
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
import { AppsList } from "./_components/apps-list";

export default function ProjectPage() {
  const params = useParams();
  const workspace = useWorkspaceNavigation();
  const projectId = typeof params?.projectId === "string" ? params.projectId : "";

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Apps</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <Button size="md" variant="primary" asChild>
            <Link href={routes.projects.apps.new({ workspaceSlug: workspace.slug, projectId })}>
              <Plus iconSize="sm-regular" />
              Create app
            </Link>
          </Button>
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        <AppsList />
      </PageBody>
    </PageContainer>
  );
}
