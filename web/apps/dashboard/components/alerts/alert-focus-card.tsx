"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { shortenId } from "@/lib/shorten-id";
import { requestDropMedianFraction } from "@unkey/clickhouse/src/alert-thresholds";
import { Badge, TimestampInfo } from "@unkey/ui";
import Link from "next/link";
import {
  alertMetricLabel,
  formatAlertDistance,
  formatAlertValue,
  hasFixedAlertThreshold,
  isErrorRateMetric,
} from "./format";
import { AlertStatusBadge } from "./status-badge";
import type { AlertDetailData } from "./types";

export function AlertFocusCard({ alert }: { alert: AlertDetailData }) {
  const workspace = useWorkspaceNavigation();
  const deploymentHref = alert.deploymentId
    ? routes.projects.apps.deployment({
        workspaceSlug: workspace.slug,
        projectId: alert.projectId,
        appId: alert.appId,
        deploymentId: alert.deploymentId,
      })
    : null;

  return (
    <section className="overflow-hidden rounded-lg border border-errorA-5 bg-gray-1 shadow-sm">
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-grayA-4 bg-errorA-2 px-5 py-4">
        <div className="flex items-center gap-3">
          <h2 className="text-sm font-semibold text-gray-12">
            Focused alert · {alertMetricLabel(alert.metric)}
          </h2>
          <AlertStatusBadge status={alert.status} />
        </div>
      </div>
      <dl
        className={
          alert.metric === "requests_drop"
            ? "grid grid-cols-1 divide-y divide-grayA-4 md:grid-cols-3 md:divide-x md:divide-y-0"
            : "grid grid-cols-2 divide-x divide-y divide-grayA-4 md:grid-cols-4 md:divide-y-0"
        }
      >
        {alert.metric === "requests_drop" ? (
          <>
            <DetailValue
              label="Observed"
              value={formatAlertValue(alert.metric, alert.observedValue)}
            />
            <DetailValue
              label="Recent 1h median"
              value={formatAlertValue(alert.metric, alert.baselineMean)}
            />
            <DetailValue
              label="Threshold"
              value={formatAlertValue(alert.metric, alert.baselineMean * requestDropMedianFraction)}
              hint="25% of recent median"
            />
          </>
        ) : (
          <>
            <DetailValue
              label="Observed"
              value={formatAlertValue(alert.metric, alert.observedValue)}
            />
            <DetailValue
              label={isErrorRateMetric(alert.metric) ? "Baseline rate" : "Baseline mean"}
              value={
                hasFixedAlertThreshold(alert.metric)
                  ? "Fixed limit"
                  : formatAlertValue(alert.metric, alert.baselineMean)
              }
            />
            <DetailValue
              label={isErrorRateMetric(alert.metric) ? "Std dev" : "Standard deviation"}
              value={
                hasFixedAlertThreshold(alert.metric)
                  ? "Not used"
                  : formatAlertValue(alert.metric, alert.baselineStddev)
              }
            />
            <DetailValue
              label="Distance"
              value={formatAlertDistance(alert.metric, alert.observedValue, alert.baselineMean)}
            />
          </>
        )}
      </dl>
      <div className="grid gap-5 border-t border-grayA-4 px-5 py-4 text-sm md:grid-cols-2">
        <div>
          <div className="mb-1 text-xs text-gray-9">Anomaly window</div>
          <div className="flex flex-wrap items-center gap-1 text-gray-12">
            <TimestampInfo value={alert.windowStart} className="underline decoration-dotted" />
            <span className="text-gray-8">to</span>
            <TimestampInfo value={alert.windowEnd} className="underline decoration-dotted" />
          </div>
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
      {alert.status === "resolved" ? (
        <div className="border-t border-successA-5 bg-successA-2 px-5 py-4">
          <div className="mb-2 flex flex-wrap items-center gap-2">
            <Badge variant="success">Resolved</Badge>
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
        </div>
      ) : null}
    </section>
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
