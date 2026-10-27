"use client";

import { MarkerGlyph } from "@/components/charts/chart-markers";
import { formatBucketInterval } from "@/components/charts/format-interval";
import { RegionFlag } from "@/components/region-flag";
import { plural } from "@/lib/fmt";
import { regionInfo } from "@/lib/regions";
import { Button, useElapsed } from "@unkey/ui";
import type { Route } from "next";
import Link from "next/link";
import type { ReactNode } from "react";
import { type CrashEpisode, EVENT_KIND } from "./events-model";
import type { LogsTarget } from "./instances-model";

export function EpisodeDetail({
  episode,
  logsHref,
  withDate,
}: {
  episode: CrashEpisode;
  logsHref: (target: LogsTarget) => Route;
  withDate: boolean;
}) {
  const elapsed = useElapsed(episode.latestMs, "short");
  const region = regionInfo(episode.region);
  return (
    <div className="flex flex-col gap-2 text-xs">
      <div className="flex flex-col gap-0.5">
        <div className="flex items-center justify-between gap-2">
          <span className="flex min-w-0 items-center gap-2">
            <MarkerGlyph shape="diamond" />
            <span className="truncate font-medium text-gray-12">
              {EVENT_KIND[episode.kind].label}
            </span>
          </span>
          <span className="shrink-0 text-gray-9">{elapsed}</span>
        </div>
        <span className="pl-4 tabular-nums text-gray-10">
          {episode.restarts > 1
            ? `${plural(episode.restarts, "restart")} · ${formatBucketInterval(episode.startMs, episode.endMs, withDate)}`
            : formatBucketInterval(episode.startMs, undefined, withDate)}
        </span>
      </div>
      <dl className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-3 gap-y-1">
        <DetailRow label="Instance">
          <span className="font-mono break-all">{episode.instance}</span>
        </DetailRow>
        <DetailRow label="Region">
          <span className="flex items-center gap-1.5">
            <RegionFlag region={episode.region} shape="rect" size="xs" />
            {region.city}
          </span>
        </DetailRow>
        {episode.exitCode !== null && (
          <DetailRow label="Exit">
            <span className="tabular-nums">{episode.exitCode}</span>
            {episode.reason && <span className="text-gray-10"> · {episode.reason}</span>}
          </DetailRow>
        )}
      </dl>
      <Button
        variant="outline"
        size="sm"
        className="mt-2 w-full"
        render={
          <Link
            href={logsHref({ deploymentId: episode.deploymentId, instanceId: episode.instanceId })}
          />
        }
      >
        View logs
      </Button>
    </div>
  );
}

function DetailRow({ label, children }: { label: string; children: ReactNode }) {
  return (
    <>
      <dt className="font-medium text-gray-10">{label}</dt>
      <dd className="min-w-0 text-gray-12">{children}</dd>
    </>
  );
}
