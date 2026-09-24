"use client";
import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import { useAppHomeHref } from "@/hooks/use-app-home-href";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
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
import { parseAsString, parseAsStringLiteral, useQueryState } from "nuqs";
import { AppCard, AppCardSkeleton } from "./app-card";
import { type AppRowData, filterApps, toAppRow } from "./app-row-model";
import { APPS_VIEWS, AppsListControls, type AppsView } from "./apps-list-controls";
import { AppsTable } from "./apps-table";

// One row at the 3-column desktop width so loading doesn't tower over the
// real list before it resolves.
const MAX_SKELETON_COUNT = 3;

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
  const [view, setView] = useQueryState(
    "view",
    parseAsStringLiteral(APPS_VIEWS).withDefault("grid").withOptions(queryOptions),
  );
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
  const project = useLiveQuery(
    (q) =>
      q.from({ project: collection.projects }).where(({ project }) => eq(project.id, projectId)),
    [projectId],
  );
  const headlineByApp = new Map(
    (project.data[0]?.apps ?? []).map((app) => [app.id, app.headlineDeployment]),
  );
  const rows = filterApps(
    apps.data.map((app) =>
      toAppRow(
        app,
        headlineByApp.get(app.id) ?? null,
        appHomeHref({ workspaceSlug: workspace.slug, projectId, appId: app.id }),
      ),
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
        isLoading={apps.isLoading || project.isLoading}
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
  if (isLoading) {
    return (
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
        {Array.from({ length: MAX_SKELETON_COUNT }).map((_, i) => (
          // biome-ignore lint/suspicious/noArrayIndexKey: skeleton items don't need stable keys
          <AppCardSkeleton key={i} />
        ))}
      </div>
    );
  }
  if (rows.length === 0) {
    return <p className="py-12 text-center text-[13px] text-gray-9">No apps match "{search}"</p>;
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
