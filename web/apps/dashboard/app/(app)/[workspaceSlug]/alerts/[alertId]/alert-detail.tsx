"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { shortenId } from "@/lib/shorten-id";
import { useWorkspace } from "@/providers/workspace-provider";
import {
  Badge,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
  TimestampInfo,
} from "@unkey/ui";
import Link from "next/link";
import {
  alertMetricLabel,
  formatAlertDistance,
  formatAlertValue,
  hasFixedAlertThreshold,
} from "../format";
import { ResolveAlertButton } from "../resolve-alert-button";
import { AlertStatusBadge } from "../status-badge";
import type { AlertDetailData, AlertTimeseriesData } from "../types";
import { AlertChart } from "./alert-chart";

export function AlertDetail({
  alert,
  timeseries,
  timeseriesLoading,
  timeseriesError,
}: {
  alert: AlertDetailData;
  timeseries: AlertTimeseriesData | undefined;
  timeseriesLoading: boolean;
  timeseriesError: boolean;
}) {
  const workspace = useWorkspaceNavigation();
  const { user } = useWorkspace();
  const appHref = routes.projects.apps.overview({
    workspaceSlug: workspace.slug,
    projectId: alert.projectId,
    appId: alert.appId,
  });
  const deploymentHref = alert.deploymentId
    ? routes.projects.apps.deployment({
        workspaceSlug: workspace.slug,
        projectId: alert.projectId,
        appId: alert.appId,
        deploymentId: alert.deploymentId,
      })
    : null;

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <div className="flex flex-wrap items-center gap-3">
            <PageHeaderTitle>{alertMetricLabel(alert.metric)}</PageHeaderTitle>
            <AlertStatusBadge status={alert.status} />
          </div>
          <PageHeaderDescription>
            <Link href={appHref} className="font-medium text-gray-12 hover:underline">
              {alert.appName}
            </Link>{" "}
            <span aria-hidden="true">›</span> {alert.environmentName}
          </PageHeaderDescription>
        </PageHeaderContent>
        <PageHeaderActions>
          <span className="text-xs text-gray-9">Fired</span>
          <TimestampInfo
            value={alert.firedAt}
            displayType="relative"
            className="text-xs underline decoration-dotted"
          />
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        <AlertChart
          metric={alert.metric}
          data={timeseries}
          loading={timeseriesLoading}
          error={timeseriesError}
        />

        <section className="overflow-hidden rounded-lg border border-grayA-4 bg-gray-1">
          <div className="border-b border-grayA-4 px-5 py-4">
            <h2 className="text-sm font-semibold text-gray-12">Alert details</h2>
            <p className="text-xs text-gray-9">Detector values captured when the alert fired.</p>
          </div>
          <dl className="grid grid-cols-1 divide-y divide-grayA-4 sm:grid-cols-2 sm:divide-x sm:divide-y-0 lg:grid-cols-4">
            <DetailValue
              label="Observed"
              value={formatAlertValue(alert.metric, alert.observedValue)}
            />
            <DetailValue
              label="Baseline mean"
              value={formatAlertValue(alert.metric, alert.baselineMean)}
            />
            <DetailValue
              label="Standard deviation"
              value={formatAlertValue(alert.metric, alert.baselineStddev)}
            />
            <DetailValue
              label="Distance"
              value={formatAlertDistance(
                alert.metric,
                alert.observedValue,
                alert.baselineMean,
                alert.baselineStddev,
              )}
              hint={
                hasFixedAlertThreshold(alert.metric)
                  ? undefined
                  : `Threshold ${alert.thresholdSigma.toFixed(1)}σ`
              }
            />
          </dl>
          <div className="grid gap-5 border-t border-grayA-4 px-5 py-4 text-sm md:grid-cols-3">
            <div>
              <div className="mb-1 text-xs text-gray-9">Anomaly window</div>
              <div className="flex flex-wrap items-center gap-1 text-gray-12">
                <TimestampInfo value={alert.windowStart} className="underline decoration-dotted" />
                <span className="text-gray-8">to</span>
                <TimestampInfo value={alert.windowEnd} className="underline decoration-dotted" />
              </div>
            </div>
            <div>
              <div className="mb-1 text-xs text-gray-9">Last anomalous window</div>
              <TimestampInfo
                value={alert.lastSeenAt}
                className="text-gray-12 underline decoration-dotted"
              />
            </div>
            <div>
              <div className="mb-1 text-xs text-gray-9">Deployment at detection</div>
              {deploymentHref && alert.deploymentId ? (
                <Link
                  href={deploymentHref}
                  className="font-mono text-xs text-gray-12 underline decoration-dotted"
                >
                  {shortenId(alert.deploymentId)}
                </Link>
              ) : (
                <span className="text-gray-9">No deployment context</span>
              )}
            </div>
          </div>
        </section>

        {alert.status === "resolved" ? (
          <section className="rounded-lg border border-successA-5 bg-successA-2 px-5 py-4">
            <div className="mb-3 flex flex-wrap items-center gap-2">
              <Badge variant="success">Resolved</Badge>
              <span className="text-xs text-gray-9">
                by {resolvedByLabel(alert.resolvedBy, user)}
              </span>
              {alert.resolvedAt ? (
                <TimestampInfo
                  value={alert.resolvedAt}
                  className="text-xs text-gray-10 underline decoration-dotted"
                />
              ) : null}
            </div>
            <p className="whitespace-pre-wrap text-sm leading-6 text-gray-12">
              {alert.resolutionMessage ?? "No resolution message was recorded."}
            </p>
          </section>
        ) : (
          <section className="flex flex-col items-start justify-between gap-4 rounded-lg border border-grayA-4 bg-gray-1 px-5 py-4 sm:flex-row sm:items-center">
            <div>
              <h2 className="text-sm font-semibold text-gray-12">Ready to close this alert?</h2>
              <p className="text-xs text-gray-9">
                Record the cause and action before resolving it.
              </p>
            </div>
            <ResolveAlertButton alertId={alert.id} />
          </section>
        )}
      </PageBody>
    </PageContainer>
  );
}

function DetailValue({ label, value, hint }: { label: string; value: string; hint?: string }) {
  return (
    <div className="px-5 py-4">
      <dt className="text-xs text-gray-9">{label}</dt>
      <dd className="mt-1 text-lg font-semibold tabular-nums text-gray-12">{value}</dd>
      {hint ? <div className="mt-0.5 text-xs text-gray-9">{hint}</div> : null}
    </div>
  );
}

function resolvedByLabel(
  resolvedBy: string | null,
  user: { id: string; fullName: string | null; email: string } | null,
): string {
  if (resolvedBy === "system") {
    return "System";
  }
  if (resolvedBy && user?.id === resolvedBy) {
    return user.fullName ?? user.email;
  }
  return resolvedBy ?? "Unknown";
}
