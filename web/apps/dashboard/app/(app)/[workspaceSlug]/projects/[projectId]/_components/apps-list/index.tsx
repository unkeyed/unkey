"use client";
import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import { ListViews } from "@/components/list-views";
import { ResourceSearchInput, useListSearch } from "@/components/resource-search-input";
import { useAppHomeHref } from "@/hooks/use-app-home-href";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { isDeploymentInFlight } from "@/lib/collections/deploy/deployment-status";
import { useCollectionPolling } from "@/lib/collections/use-collection-polling";
import { refetchProjectApps, useAppsInProject } from "@/lib/deploy-data";
import { GRID_PAGE_SIZE } from "@/lib/list-limits";
import { useShowMore } from "@/lib/loadable/list";
import { routes } from "@/lib/navigation/routes";
import { IconCubeOutline18, IconPlusOutline18 } from "@unkey/icons";
import {
  Button,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
} from "@unkey/ui";
import { useParams, useRouter } from "next/navigation";
import { AppCard, AppCardSkeleton } from "./app-card";
import { filterApps, toAppRow } from "./app-row-model";
import { APPS_TABLE, AppsTableRow } from "./apps-table";

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

  const apps = useAppsInProject(projectId);
  const hasInFlightDeployment = apps.data.some(
    (app) => app.headlineDeployment && isDeploymentInFlight(app.headlineDeployment.status),
  );
  useCollectionPolling(() => refetchProjectApps(projectId), {
    intervalMs: hasInFlightDeployment ? BUILDING_POLL_MS : IDLE_POLL_MS,
    enabled: true,
  });
  const list = useShowMore({
    all: apps.data.map((app) =>
      toAppRow(app, appHomeHref({ workspaceSlug: workspace.slug, projectId, appId: app.id })),
    ),
    isLoading: apps.isLoading,
    matches: (rows) => filterApps(rows, search),
    search,
    pageSize: GRID_PAGE_SIZE,
  });

  return (
    <>
      <ListViews
        list={list}
        viewStorageKey="apps-view"
        noun="apps"
        failedTitle="Could not load apps"
        empty={<NoApps onCreate={openCreateApp} />}
        controls={
          <ResourceSearchInput
            label="Search apps"
            placeholder="Search by name, repo, image or domain"
          />
        }
        table={APPS_TABLE}
        rowKey={(row) => row.app.id}
        renderCard={(row) => <AppCard row={row} projectId={projectId} />}
        renderRow={(row) => <AppsTableRow row={row} projectId={projectId} />}
        cardSkeleton={<AppCardSkeleton />}
      />
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
