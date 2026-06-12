"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
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
          <PageHeaderTitle>Overview</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <Button size="md" variant="primary" asChild>
            <Link href={`/${workspace.slug}/projects/${projectId}/apps/new`}>
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
