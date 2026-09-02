"use client";
import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
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
import { IconPlusOutline18 } from "nucleo-ui-outline-18";
import { AppsList } from "./_components/apps-list";

export default function ProjectPage() {
  const params = useParams();
  const workspace = useWorkspaceNavigation();
  const projectId = typeof params?.projectId === "string" ? params.projectId : "";
  const { gated, openPaywall, planGate } = useDeployActionGate();

  return (
    <PageContainer className="flex-1">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Apps</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          {/* Without a Compute plan the button opens the paywall instead of the
              new-app wizard. */}
          {gated ? (
            <Button size="md" variant="primary" onClick={openPaywall}>
              <IconPlusOutline18 />
              Create app
            </Button>
          ) : (
            <Button
              size="md"
              variant="primary"
              render={
                <Link
                  href={routes.projects.apps.new({ workspaceSlug: workspace.slug, projectId })}
                />
              }
            >
              <IconPlusOutline18 />
              Create app
            </Button>
          )}
        </PageHeaderActions>
      </PageHeader>
      <PageBody className="flex-1">
        <AppsList />
      </PageBody>
      {planGate}
    </PageContainer>
  );
}
