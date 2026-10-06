"use client";

import { ProjectDataProvider } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import { RectFlag } from "@/app/(app)/[workspaceSlug]/projects/_components/region/rect-flag";
import { regionInfo } from "@/app/(app)/[workspaceSlug]/projects/_components/region/region-info";
import { collection } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { Skeleton } from "@unkey/ui";
import type { ReactNode } from "react";
import { useApp, useNewAppFlow } from "../flow";
import { type SourceKind, sourceCopy } from "../wizard-model";
import { useFirstDeploy } from "./deploy/use-first-deploy";
import { PaneSubmit } from "./pane-actions";
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

function ReviewRow({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex items-start gap-3 border-t border-grayA-3 py-2.5 first:border-t-0">
      <span className="shrink-0 text-sm font-medium text-gray-11">{label}</span>
      <span className="ml-auto flex min-w-0 flex-col items-end text-right">{children}</span>
    </div>
  );
}

function Plain({ children }: { children: ReactNode }) {
  return <span className="max-w-full truncate text-sm text-gray-12">{children}</span>;
}

function Mono({ children }: { children: ReactNode }) {
  return <span className="max-w-full truncate font-mono text-xs text-gray-12">{children}</span>;
}

const loading = <Skeleton className="h-4 w-24 rounded" />;

function RegionList({ names }: { names: string[] }) {
  if (names.length === 0) {
    return <Plain>None</Plain>;
  }
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
  const app = useApp(projectId, appId);
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

  const connection = github?.repoConnection;
  const production = settings.status === "ready" ? settings.production : null;
  const variableCount = new Set(variables.map((v) => v.key)).size;
  const origin = (): ReactNode => {
    if (source === "oci") {
      if (!app) {
        return loading;
      }
      return app.imageReference ? <Mono>{app.imageReference}</Mono> : <Plain>None</Plain>;
    }
    if (github === undefined) {
      return loading;
    }
    if (!connection) {
      return <Plain>None</Plain>;
    }
    return (
      <Mono>
        {connection.repositoryFullName} · {connection.defaultBranch ?? github.defaultBranch}
      </Mono>
    );
  };

  return (
    <div className="flex flex-1 flex-col gap-4">
      <div className="flex flex-col">
        <ReviewRow label="Source">
          <Plain>{sourceCopy[source].sourceLabel}</Plain>
        </ReviewRow>
        <ReviewRow label={sourceCopy[source].originLabel}>{origin()}</ReviewRow>
        <ReviewRow label="App name">{app ? <Plain>{app.name}</Plain> : loading}</ReviewRow>
        <ReviewRow label="Port">{production ? <Mono>{production.port}</Mono> : loading}</ReviewRow>
        <ReviewRow label="Regions">
          {production ? <RegionList names={production.regions.map((r) => r.name)} /> : loading}
        </ReviewRow>
        <ReviewRow label="Size">
          {production ? (
            <span className="flex items-center gap-2.5">
              <SizeBadge size={production} />
              <Spec size={production} />
            </span>
          ) : (
            loading
          )}
        </ReviewRow>
        <ReviewRow label="Environment variables">
          <Plain>{variableCount === 0 ? "None" : `${variableCount} set`}</Plain>
        </ReviewRow>
      </div>
      <PaneSubmit
        loading={deploy.isDeploying}
        disabled={!deploy.canDeploy}
        onClick={() => (gated ? openPaywall() : deploy.start())}
      >
        Deploy
      </PaneSubmit>
      {planGate}
    </div>
  );
}
