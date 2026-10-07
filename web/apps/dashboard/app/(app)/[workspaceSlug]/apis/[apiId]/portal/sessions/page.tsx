"use client";

import { useApiKeyAuthId } from "@/hooks/use-api-key-auth-id";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { useUpdatePortal } from "@/lib/portal/use-portal";
import { useWorkspace } from "@/providers/workspace-provider";
import { match } from "@unkey/match";
import {
  EmptyState,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateTitle,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
} from "@unkey/ui";
import { redirect, useParams } from "next/navigation";
import { use } from "react";
import {
  DisabledBanner,
  PortalErrorPanel,
  usePortalSurfaceState,
} from "../components/portal-lifecycle-page";
import { PortalSessions, PortalSessionsSkeleton } from "../components/portal-sessions";

type Props = {
  params: Promise<{ apiId: string }>;
};

export default function PortalSessionsPage(props: Props) {
  const { apiId } = use(props.params);
  const workspace = useWorkspaceNavigation();
  const { projectId } = useParams<{ projectId?: string }>();
  const {
    keyAuthId,
    isLoading: keyAuthIdLoading,
    isError: keyAuthIdError,
    refetch: refetchKeyAuthId,
  } = useApiKeyAuthId(apiId);
  const { state, retry } = usePortalSurfaceState(
    keyAuthId,
    keyAuthIdLoading,
    keyAuthIdError,
    refetchKeyAuthId,
  );
  const updatePortal = useUpdatePortal(keyAuthId ?? "");
  const { user } = useWorkspace();
  const isAdmin = user?.role === "admin";

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Sessions</PageHeaderTitle>
          <PageHeaderDescription>
            Users with a portal session that has not expired or been revoked.
          </PageHeaderDescription>
        </PageHeaderContent>
      </PageHeader>
      <PageBody>
        {match(state)
          .with({ status: "loading" }, () => <PortalSessionsSkeleton />)
          .with({ status: "enabled" }, ({ portal }) =>
            isAdmin ? (
              <PortalSessions portalId={portal.id} canRevoke={isAdmin} />
            ) : (
              <EmptyState>
                <EmptyStateHeader>
                  <EmptyStateTitle>Sessions are admin-only</EmptyStateTitle>
                  <EmptyStateDescription>
                    Ask a workspace admin to see or revoke who is signed in to this portal.
                  </EmptyStateDescription>
                </EmptyStateHeader>
              </EmptyState>
            ),
          )
          .with({ status: "disabled" }, ({ portal }) => (
            <DisabledBanner
              description="All sessions ended when the portal was disabled. Your users need a new link from your app after you re-enable it."
              enabling={updatePortal.isLoading}
              onEnable={() => updatePortal.mutate({ portal: portal.id, enabled: true })}
            />
          ))
          .with({ status: "notConfigured" }, () =>
            redirect(routes.apis.portal({ workspaceSlug: workspace.slug, projectId, apiId })),
          )
          .with({ status: "error" }, ({ message }) => (
            <PortalErrorPanel message={message} onRetry={retry} />
          ))
          .exhaustive()}
      </PageBody>
    </PageContainer>
  );
}
