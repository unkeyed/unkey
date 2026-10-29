"use client";

import { formatSpan } from "@/components/charts/format-interval";
import { RegionFlag } from "@/components/region-flag";
import { plural } from "@/lib/fmt";
import { regionInfo } from "@/lib/regions";
import { Button, useElapsed } from "@unkey/ui";
import type { Route } from "next";
import Link from "next/link";
import type { ReactNode } from "react";
import {
  type CrashEpisode,
  EVENT_KIND,
  type EpisodeGroup,
  type ErrorGroup,
  episodeError,
  episodeLogs,
  spanEnd,
} from "./events-model";
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
  const error = episodeError(episode);
  const range = formatSpan(episode.startMs, spanEnd([episode]), withDate);
  return (
    <div className="flex flex-col gap-2 text-xs">
      <div className="flex flex-col gap-0.5">
        <div className="flex items-center justify-between gap-2">
          <span className="min-w-0 truncate font-medium text-gray-12">
            {EVENT_KIND[episode.kind].label}
          </span>
          <span className="shrink-0 text-gray-9">{elapsed}</span>
        </div>
        <span className="tabular-nums text-gray-10">
          {episode.restarts > 1 ? `${plural(episode.restarts, "restart")} · ${range}` : range}
        </span>
      </div>
      <div className="flex flex-col gap-1">
        <span className="break-words font-medium text-gray-12">{error.title}</span>
        {error.detail && (
          <span className="line-clamp-3 whitespace-pre-line break-all text-gray-10">
            {error.detail}
          </span>
        )}
      </div>
      <dl className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-3 gap-y-1">
        <DetailRow label="Region">
          <span className="flex items-center gap-1.5">
            <RegionFlag region={episode.region} shape="rect" size="xs" />
            {region.city}
          </span>
        </DetailRow>
        {error.exit && (
          <DetailRow label="Exit">
            <span className="tabular-nums">{error.exit.code}</span>
            {error.exit.reason && <span className="text-gray-10"> · {error.exit.reason}</span>}
          </DetailRow>
        )}
        <DetailRow label="Instance">
          <span className="break-all">{episode.instance}</span>
        </DetailRow>
      </dl>
      <Button
        variant="outline"
        size="sm"
        className="mt-2 w-full text-xs"
        render={<Link href={logsHref(episodeLogs([episode]))} />}
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

export function EpisodeGroupDetail({
  group,
  logsHref,
  withDate,
}: {
  group: EpisodeGroup;
  logsHref: (target: LogsTarget) => Route;
  withDate: boolean;
}) {
  return (
    <div className="flex flex-col gap-2 text-xs">
      <span className="flex items-baseline gap-1.5">
        <span className="font-medium text-gray-12">Crash events</span>
        <span className="tabular-nums text-gray-10">{group.episodes.length}</span>
      </span>
      <ul className="-mx-3 flex max-h-56 flex-col overflow-y-auto border-y border-grayA-4 px-1.5 py-1">
        {group.errors.map((error) => (
          <li key={error.title}>
            <ErrorRow error={error} href={logsHref(error.logs)} />
          </li>
        ))}
      </ul>
      <span className="tabular-nums text-gray-10">
        {formatSpan(group.span.startMs, spanEnd(group.episodes), withDate)}
      </span>
      <Button
        variant="outline"
        size="sm"
        className="w-full text-xs"
        render={<Link href={logsHref(group.logs)} />}
      >
        View logs
      </Button>
    </div>
  );
}

function ErrorRow({ error, href }: { error: ErrorGroup; href: Route }) {
  const kinds = error.kinds.map((kind) => EVENT_KIND[kind].label).join(", ");
  const cities = error.regions.map((region) => regionInfo(region).city).join(", ");
  return (
    <Button
      variant="ghost"
      size="sm"
      className="h-auto w-full items-start justify-start gap-2 rounded-md px-1.5 py-1.5 text-xs font-normal"
      render={<Link href={href} />}
    >
      <span className="flex min-w-0 flex-1 flex-col gap-0.5 text-left">
        <span className="flex items-baseline gap-2">
          <span title={error.title} className="min-w-0 flex-1 truncate font-medium text-gray-12">
            {error.title}
          </span>
          <span className="shrink-0 tabular-nums text-gray-10">×{error.count}</span>
        </span>
        <span className="truncate text-gray-10">
          {kinds} · {cities}
        </span>
      </span>
    </Button>
  );
}
