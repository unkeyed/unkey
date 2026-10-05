"use client";

import { ProjectDataProvider } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import { RectFlag } from "@/app/(app)/[workspaceSlug]/projects/_components/region/rect-flag";
import { regionInfo } from "@/app/(app)/[workspaceSlug]/projects/_components/region/region-info";
import { collection } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { match } from "@unkey/match";
import { Button, Skeleton } from "@unkey/ui";
import { useNewAppFlow } from "../flow";
import type { SourceKind } from "../wizard-model";
import { type ReviewRow, type ReviewValue, reviewRows } from "./deploy/review-rows";
import { useFirstDeploy } from "./deploy/use-first-deploy";
import { PaneActions } from "./pane-actions";
import { useAppSettings } from "./settings";
import { SizeBadge, Spec } from "./settings/size-field";

type DeployPaneProps = { appId: string; source: SourceKind };

export function DeployPane(props: DeployPaneProps) {
  const { projectId } = useNewAppFlow();
  return (
    <ProjectDataProvider projectId={projectId} appId={props.appId}>
      <Review {...props} />
    </ProjectDataProvider>
  );
}

function ReviewRowView({ row }: { row: ReviewRow }) {
  return (
    <div className="flex items-start gap-3 border-t border-grayA-3 py-2.5 first:border-t-0">
      <span className="shrink-0 text-sm font-medium text-gray-11">{row.label}</span>
      <span className="ml-auto flex min-w-0 flex-col items-end text-right">
        <ReviewValueView value={row.value} />
      </span>
    </div>
  );
}

function ReviewValueView({ value }: { value: ReviewValue }) {
  return match(value)
    .with({ type: "loading" }, () => <Skeleton className="h-4 w-24 rounded" />)
    .with({ type: "text" }, ({ text }) => (
      <span className="max-w-full truncate text-sm text-gray-12">{text}</span>
    ))
    .with({ type: "mono" }, ({ text }) => (
      <span className="max-w-full truncate font-mono text-xs text-gray-12">{text}</span>
    ))
    .with({ type: "size" }, ({ size }) => (
      <span className="flex items-center gap-2.5">
        <SizeBadge size={size} />
        <Spec size={size} />
      </span>
    ))
    .with({ type: "regions" }, ({ names }) => <RegionList names={names} />)
    .exhaustive();
}

function RegionList({ names }: { names: string[] }) {
  return (
    <span className="flex flex-col items-end gap-1">
      {names.map((name) => {
        const region = regionInfo(name);
        return (
          <span key={name} className="flex items-center gap-2 text-sm text-gray-12">
            <RectFlag flag={region.flag} size="sm" />
            {region.city}
          </span>
        );
      })}
    </span>
  );
}

function Review({ appId, source }: DeployPaneProps) {
  const { projectId, dispatch } = useNewAppFlow();
  const { gated, openPaywall, planGate } = useDeployActionGate();
  const settings = useAppSettings(projectId, appId);
  const deploy = useFirstDeploy(projectId, appId, source, (deploymentId) =>
    dispatch({ type: "deployment-created", deploymentId }),
  );
  const { data: apps } = useLiveQuery(
    (q) =>
      q
        .from({ app: collection.apps })
        .where(({ app }) => and(eq(app.projectId, projectId), eq(app.id, appId))),
    [projectId, appId],
  );
  const { data: variables } = useLiveQuery(
    (q) =>
      q
        .from({ v: collection.envVars })
        .where(({ v }) => and(eq(v.projectId, projectId), eq(v.appId, appId))),
    [projectId, appId],
  );
  const { data: github } = trpc.github.getInstallations.useQuery(
    { projectId, appId },
    { enabled: source === "git" },
  );

  const app = apps.at(0);
  const connection = github?.repoConnection;
  const rows = reviewRows({
    source,
    app,
    repository:
      github === undefined
        ? undefined
        : connection
          ? {
              fullName: connection.repositoryFullName,
              branch: connection.defaultBranch ?? github.defaultBranch,
            }
          : null,
    runtime:
      settings.status === "ready"
        ? {
            port: settings.production.port,
            regions: settings.production.regions.map((r) => r.name),
            size: {
              cpuMillicores: settings.production.cpuMillicores,
              memoryMib: settings.production.memoryMib,
            },
          }
        : undefined,
    variableCount: new Set(variables.map((v) => v.key)).size,
  });

  return (
    <div className="flex flex-1 flex-col gap-4">
      <div className="flex flex-col">
        {rows.map((row) => (
          <ReviewRowView key={row.label} row={row} />
        ))}
      </div>
      <PaneActions>
        <Button
          type="button"
          variant="primary"
          size="sm"
          className="px-3"
          loading={deploy.isDeploying}
          disabled={!deploy.canDeploy}
          onClick={() => (gated ? openPaywall() : deploy.start())}
        >
          Deploy
        </Button>
      </PaneActions>
      {planGate}
    </div>
  );
}
