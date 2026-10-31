"use client";

import { RegionFlag } from "@/components/region-flag";
import type { InstanceRow } from "@/lib/metrics/app-metrics";
import { regionInfo } from "@/lib/regions";
import { IconChevronRightOutline12, IconServerOutline18 } from "@unkey/icons";
import {
  ResourceListBody,
  ResourceListContent,
  ResourceListItem,
  ResourceListRow,
  TimestampInfo,
} from "@unkey/ui";
import { cn } from "cn";
import type { Route } from "next";
import Link from "next/link";
import type { ReactNode } from "react";
import {
  type InstanceTableRow,
  LINE_COLOR,
  type LineTone,
  type LogsTarget,
  type UsageCell,
  instanceIdsOf,
  instanceSrLabel,
} from "./instances-model";

const COLUMNS =
  "grid grid-cols-[minmax(0,1.3fr)_minmax(0,1.2fr)_minmax(0,0.8fr)_88px_88px_72px] items-center gap-4 px-4";

const TONE_TEXT: Record<LineTone, { name: string; status: string }> = {
  unhealthy: { name: "text-gray-12", status: "text-error-11" },
  previous: { name: "text-gray-12", status: "text-gray-11" },
  current: { name: "text-gray-12", status: "text-gray-11" },
  stopped: { name: "text-gray-11", status: "text-gray-11" },
};

export function InstanceTable({
  rows,
  logsHref,
  onOpen,
  onHover,
  footer,
}: {
  rows: InstanceTableRow[];
  logsHref: (target: LogsTarget) => Route;
  onOpen: (row: InstanceRow) => void;
  onHover: (key: string | null) => void;
  footer?: ReactNode;
}) {
  return (
    <ResourceListContent className="rounded-none border-0 border-t border-grayA-4 bg-transparent">
      <div className={cn(COLUMNS, "h-8 border-b border-grayA-4 bg-gray-2 text-xs text-gray-10")}>
        <span>Instance</span>
        <span>Status</span>
        <span>Region</span>
        <span>CPU</span>
        <span>Memory</span>
        <span />
      </div>
      <ResourceListBody>
        {rows.map(({ instance, usage }) => {
          const { tone } = instance;
          return (
            <ResourceListItem key={instance.key}>
              <ResourceListRow
                label={`Open ${instance.key}, ${instanceSrLabel(instance)}`}
                onActivate={() => onOpen(instance)}
                onPointerEnter={() => onHover(instance.key)}
                onPointerLeave={() => onHover(null)}
                onFocus={() => onHover(instance.key)}
                onBlur={() => onHover(null)}
                className={cn(COLUMNS, "group/row h-12 text-sm")}
              >
                <span className="flex min-w-0 items-center gap-2">
                  <span aria-hidden="true" className="relative flex shrink-0">
                    <IconServerOutline18 className="size-4 text-gray-12 [mask-image:radial-gradient(circle_at_calc(100%-3px)_calc(100%-3px),transparent_4.5px,#000_5px)]" />
                    <span
                      className="absolute right-0 bottom-0 size-1.5 rounded-full"
                      style={{ backgroundColor: LINE_COLOR[tone] }}
                    />
                  </span>
                  <span className={cn("truncate font-medium", TONE_TEXT[tone].name)}>
                    {instance.key}
                  </span>
                </span>
                <span
                  className={cn("flex min-w-0 gap-1 text-xs", TONE_TEXT[tone].status)}
                  title={instance.statusText}
                >
                  <span className="truncate">{instance.statusText}</span>
                  {instance.presence.kind === "stopped" && (
                    <TimestampInfo
                      value={instance.presence.lastSeenMs}
                      displayType="relative"
                      className="relative truncate"
                    />
                  )}
                </span>
                <span className="flex min-w-0 items-center gap-1.5 text-xs text-gray-11">
                  <RegionFlag region={instance.region.name} shape="rect" size="xs" />
                  <span className="truncate">{regionInfo(instance.region.name).city}</span>
                </span>
                <Load cell={usage.cpu} />
                <Load cell={usage.memory} />
                <span className="flex items-center justify-end gap-3">
                  {tone === "unhealthy" && (
                    <Link
                      href={logsHref({
                        deploymentIds: [instance.deployment.id],
                        instanceIds: instanceIdsOf(instance),
                      })}
                      aria-label={`Logs for ${instance.key}`}
                      className="relative rounded-sm text-xs text-gray-11 outline-none hover:text-gray-12 focus-visible:ring-1 focus-visible:ring-grayA-12"
                    >
                      Logs
                    </Link>
                  )}
                  <span className="flex items-center justify-center rounded-sm p-0.5 text-gray-9 transition-colors group-hover/row:bg-grayA-4 group-hover/row:text-gray-12 group-has-focus-visible/row:bg-grayA-4 group-has-focus-visible/row:text-gray-12 motion-reduce:transition-none">
                    <IconChevronRightOutline12 className="size-2.5" />
                  </span>
                </span>
              </ResourceListRow>
            </ResourceListItem>
          );
        })}
      </ResourceListBody>
      {footer}
    </ResourceListContent>
  );
}

function Load({ cell }: { cell: UsageCell }) {
  if (cell.percent === null) {
    return <span className="pl-1.5 font-mono text-xs text-gray-9">—</span>;
  }
  return (
    <span className="relative flex h-6 w-16 items-center px-1.5">
      <span
        aria-hidden
        className={cn(
          "absolute inset-y-0 left-0 rounded-md",
          cell.hot ? "bg-errorA-3" : "bg-grayA-3",
        )}
        style={{ width: `${Math.min(cell.percent, 100)}%` }}
      />
      <span
        className={cn(
          "relative font-mono text-xs tabular-nums",
          cell.hot ? "text-error-11" : "text-gray-12",
        )}
      >
        {cell.percent}%
      </span>
    </span>
  );
}
