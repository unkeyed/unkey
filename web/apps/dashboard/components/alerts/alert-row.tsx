"use client";

import { cn } from "@/lib/utils";
import { ResourceListItem, TimestampInfo } from "@unkey/ui";
import type { Route } from "next";
import Link from "next/link";
import { AlertRowChart } from "./alert-row-chart";
import {
  alertMetricLabel,
  formatAlertDistance,
  formatAlertValue,
  formatSigma,
  hasFixedAlertThreshold,
} from "./format";
import { ResolveAlertButton } from "./resolve-alert-button";
import { AlertStatusBadge } from "./status-badge";
import type { AlertListItem } from "./types";

export function AlertRow({
  alert,
  href,
  onSelect,
  selected = false,
}: {
  alert: AlertListItem;
  href?: Route;
  onSelect?: () => void;
  selected?: boolean;
}) {
  const chart = (
    <AlertRowChart
      appId={alert.appId}
      environmentId={alert.environmentId}
      metric={alert.metric}
      windowStart={alert.windowStart}
      windowEnd={alert.windowEnd}
    />
  );

  return (
    <ResourceListItem
      className={cn(
        "group relative flex flex-col gap-3 px-4 py-3 transition-colors hover:bg-grayA-2 lg:flex-row lg:items-center lg:gap-0",
        selected && "bg-errorA-2 ring-1 ring-inset ring-errorA-6",
      )}
    >
      {href ? (
        <Link
          href={href}
          className="absolute inset-0 z-10 rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-grayA-7"
          aria-label={`${alertMetricLabel(alert.metric)} alert for ${alert.appName}`}
        />
      ) : onSelect ? (
        <button
          type="button"
          className="absolute inset-0 z-10 rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-grayA-7"
          onClick={onSelect}
          aria-label={`Show ${alertMetricLabel(alert.metric)} alert on the chart`}
        />
      ) : null}
      <div className="flex min-w-0 flex-col gap-1 lg:w-[23%] lg:shrink-0">
        <span className="truncate text-[13px] font-semibold text-accent-12">
          {alertMetricLabel(alert.metric)}
        </span>
        <span className="truncate text-xs text-gray-9">
          {alert.appName} <span aria-hidden="true">›</span> {alert.environmentName}
        </span>
      </div>
      <div className="lg:w-[13%] lg:shrink-0">
        <AlertStatusBadge status={alert.status} />
      </div>
      <div className="flex min-w-0 flex-col gap-1 lg:w-[27%] lg:shrink-0">
        <span className="truncate text-[13px] font-medium tabular-nums text-accent-12">
          {hasFixedAlertThreshold(alert.metric) ? (
            formatAlertDistance(
              alert.metric,
              alert.observedValue,
              alert.baselineMean,
              alert.baselineStddev,
            )
          ) : alert.metric === "requests_drop" ? (
            <>
              {formatAlertValue(alert.metric, alert.observedValue)}
              <span className="font-normal text-gray-9">
                {" "}
                vs {formatAlertValue(alert.metric, alert.baselineMean)} recent (
                {formatAlertDistance(
                  alert.metric,
                  alert.observedValue,
                  alert.baselineMean,
                  alert.baselineStddev,
                )}
                )
              </span>
            </>
          ) : (
            <>
              {formatAlertValue(alert.metric, alert.observedValue)}
              <span className="font-normal text-gray-9">
                {" "}
                vs {formatAlertValue(alert.metric, alert.baselineMean)} avg
              </span>
            </>
          )}
        </span>
        {hasFixedAlertThreshold(alert.metric) || alert.metric === "requests_drop" ? null : (
          <span className="text-xs font-medium tabular-nums text-error-11">
            {formatSigma(alert.observedValue, alert.baselineMean, alert.baselineStddev)}
          </span>
        )}
      </div>
      <div className="relative z-20 lg:w-[14%] lg:shrink-0">
        <TimestampInfo
          value={alert.firedAt}
          displayType="relative"
          className="text-xs text-gray-9 underline decoration-dotted"
        />
      </div>
      <div className="flex items-center gap-3 lg:w-[23%] lg:shrink-0 lg:justify-end">
        {href ? (
          <Link
            href={href}
            className="relative z-20 rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-grayA-7"
            aria-label={`Open ${alertMetricLabel(alert.metric)} alert chart`}
          >
            {chart}
          </Link>
        ) : onSelect ? (
          <button
            type="button"
            className="relative z-20 rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-grayA-7"
            onClick={onSelect}
            aria-label={`Show ${alertMetricLabel(alert.metric)} alert chart`}
          >
            {chart}
          </button>
        ) : (
          chart
        )}
        {alert.status === "open" ? <ResolveAlertButton alertId={alert.id} /> : null}
      </div>
    </ResourceListItem>
  );
}
