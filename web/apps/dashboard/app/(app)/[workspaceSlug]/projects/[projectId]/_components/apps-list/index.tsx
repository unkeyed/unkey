"use client";
import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import { useAppHomeHref } from "@/hooks/use-app-home-href";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { isDeploymentInFlight } from "@/lib/collections/deploy/deployment-status";
import { useCollectionPolling } from "@/lib/collections/use-collection-polling";
import { routes } from "@/lib/navigation/routes";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { IconCubeOutline18, IconPlusOutline18 } from "@unkey/icons";
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
import { parseAsString, useQueryState } from "nuqs";
import { AppCard, AppCardSkeleton } from "./app-card";
import { type AppRowData, filterApps, toAppRow } from "./app-row-model";
import { AppsListControls } from "./apps-list-controls";
import { AppsTable, AppsTableSkeleton } from "./apps-table";
import { type AppsView, useAppsView } from "./use-apps-view";

// One row at the 3-column desktop width so loading doesn't tower over the
// real list before it resolves.
const SKELETON_KEYS = ["skeleton-1", "skeleton-2", "skeleton-3"];

const IDLE_POLL_MS = 60_000;
const BUILDING_POLL_MS = 5_000;

const queryOptions = { history: "replace", shallow: true, clearOnDefault: true } as const;

export function AppsList() {
  const params = useParams();
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const appHomeHref = useAppHomeHref();
  const projectId = typeof params?.projectId === "string" ? params.projectId : "";
  const { gated, openPaywall, planGate } = useDeployActionGate();
  const [search, setSearch] = useQueryState(
    "search",
    parseAsString.withDefault("").withOptions(queryOptions),
  );
  const [view, setView] = useAppsView();
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

  const apps = useLiveQuery(
    (q) => q.from({ app: collection.apps }).where(({ app }) => eq(app.projectId, projectId)),
    [projectId],
  );
  const hasInFlightDeployment = apps.data.some(
    (app) => app.headlineDeployment && isDeploymentInFlight(app.headlineDeployment.status),
  );
  useCollectionPolling(() => collection.apps.utils.refetch(), {
    intervalMs: hasInFlightDeployment ? BUILDING_POLL_MS : IDLE_POLL_MS,
    enabled: true,
  });
  const rows = filterApps(
    apps.data.map((app) =>
      toAppRow(app, appHomeHref({ workspaceSlug: workspace.slug, projectId, appId: app.id })),
    ),
    search,
  );

  if (!apps.isLoading && apps.data.length === 0) {
    return (
      <>
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
            <Button variant="primary" size="md" onClick={openCreateApp}>
              <IconPlusOutline18 />
              Create app
            </Button>
          </EmptyStateActions>
        </EmptyState>
        {planGate}
      </>
    );
  }

  return (
    <ResourceList>
      <AppsListControls
        search={search}
        onSearchChange={(value) => setSearch(value || null)}
        view={view}
        onViewChange={setView}
      />
      <AppsListBody
        isLoading={apps.isLoading}
        rows={rows}
        search={search}
        view={view}
        projectId={projectId}
      />
      {planGate}
    </ResourceList>
  );
}

function AppsListBody({
  isLoading,
  rows,
  search,
  view,
  projectId,
}: {
  isLoading: boolean;
  rows: AppRowData[];
  search: string;
  view: AppsView;
  projectId: string;
}) {
  if (isLoading && view === "list") {
    return <AppsTableSkeleton />;
  }
  if (isLoading) {
    return (
      <div aria-busy="true" className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
        {SKELETON_KEYS.map((key) => (
          <AppCardSkeleton key={key} />
        ))}
      </div>
    );
  }
  if (rows.length === 0) {
    return <p className="py-12 text-center text-sm text-gray-9">No apps match "{search}"</p>;
  }
  if (view === "list") {
    return <AppsTable rows={rows} projectId={projectId} />;
  }
  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
      {rows.map((row) => (
        <AppCard key={row.app.id} row={row} projectId={projectId} />
      ))}
    </div>
  );
}
