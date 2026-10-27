"use client";
import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import { ListViews } from "@/components/list-views";
import { useListSearch } from "@/components/resource-search-input";
import { useAppHomeHref } from "@/hooks/use-app-home-href";
import { useListView } from "@/hooks/use-list-view";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { refetchProjectApps } from "@/lib/collections/deploy/apps";
import { isDeploymentInFlight } from "@/lib/collections/deploy/deployment-status";
import { useCollectionPolling } from "@/lib/collections/use-collection-polling";
import { useShowMore } from "@/lib/loadable/list";
import { routes } from "@/lib/navigation/routes";
import { useLiveQuery } from "@tanstack/react-db";
import { IconCubeOutline18, IconPlusOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import {
  Button,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
  ResourceList,
} from "@unkey/ui";
import { useParams, useRouter } from "next/navigation";
import { AppCard, AppCardSkeleton } from "./app-card";
import { filterApps, toAppRow } from "./app-row-model";
import { AppsListControls } from "./apps-list-controls";
import { APPS_TABLE, AppsTableRow } from "./apps-table";
import { projectAppsQueryFor } from "./queries";

const IDLE_POLL_MS = 60_000;
const BUILDING_POLL_MS = 5_000;

export function AppsList() {
  const params = useParams();
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const appHomeHref = useAppHomeHref();
  const projectId = typeof params?.projectId === "string" ? params.projectId : "";
  const { gated, openPaywall, planGate } = useDeployActionGate();
  const search = useListSearch();
  const [view, setView] = useListView("apps-view");
  // Without a Compute plan, creating an app opens the paywall instead.
  const openCreateApp = () =>
    gated
      ? openPaywall()
      : router.push(
          routes.projects.apps.new({
            workspaceSlug: workspace.slug,
            projectId,
          }),
        );

  const apps = useLiveQuery(projectAppsQueryFor(projectId), [projectId]);
  const hasInFlightDeployment = apps.data.some(
    (app) => app.headlineDeployment && isDeploymentInFlight(app.headlineDeployment.status),
  );
  useCollectionPolling(() => refetchProjectApps(projectId), {
    intervalMs: hasInFlightDeployment ? BUILDING_POLL_MS : IDLE_POLL_MS,
    enabled: true,
  });
  const rows = filterApps(
    apps.data.map((app) =>
      toAppRow(app, appHomeHref({ workspaceSlug: workspace.slug, projectId, appId: app.id })),
    ),
    search,
  );
  const list = useShowMore({
    rows,
    total: apps.data.length,
    isLoading: apps.isLoading,
    search,
  });

  return (
    <>
      {match(list)
        .with({ status: "empty" }, () => <NoApps onCreate={openCreateApp} />)
        .otherwise((content) => (
          <ResourceList>
            <AppsListControls view={view} onViewChange={setView} />
            <ListViews
              view={view}
              content={content}
              noun="apps"
              table={APPS_TABLE}
              rowKey={(row) => row.app.id}
              renderCard={(row) => <AppCard row={row} projectId={projectId} />}
              renderRow={(row, table) => (
                <AppsTableRow row={row} projectId={projectId} table={table} />
              )}
              cardSkeleton={<AppCardSkeleton />}
            />
          </ResourceList>
        ))}
      {planGate}
    </>
  );
}

function NoApps({ onCreate }: { onCreate: () => void }) {
  return (
    <EmptyState>
      <EmptyStateIcon>
        <IconCubeOutline18 />
      </EmptyStateIcon>
      <EmptyStateHeader>
        <EmptyStateTitle>No Apps Found</EmptyStateTitle>
        <EmptyStateDescription>
          This project has no apps yet. Create an app to start deploying.
        </EmptyStateDescription>
      </EmptyStateHeader>
      <EmptyStateActions>
        <Button variant="primary" size="sm" onClick={onCreate}>
          <IconPlusOutline18 />
          Create app
        </Button>
      </EmptyStateActions>
    </EmptyState>
  );
}
