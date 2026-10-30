"use client";

import type { ChartIncident, ChartIncidents } from "@/components/charts/incident-markers";
import { match } from "@unkey/match";
import type { Route } from "next";
import { EpisodeDetail, EpisodeGroupDetail } from "./event-detail";
import {
  type CrashEpisode,
  type EpisodeGroup,
  crashIncidents,
  episodeGroup,
  incidentOf,
} from "./events-model";
import type { LogsTarget } from "./instances-model";

export function withCrashIncidents(
  { active, onActiveChange }: Pick<ChartIncidents, "active" | "onActiveChange">,
  episodes: CrashEpisode[],
  detail: { logsHref: (target: LogsTarget) => Route; withDate: boolean },
): ChartIncidents {
  const toChartIncident = (group: EpisodeGroup): ChartIncident => {
    const incident = incidentOf(group);
    return {
      id: incident.id,
      fromMs: incident.span.startMs,
      toMs: incident.span.endMs,
      label: incident.label,
      content: match(incident)
        .with({ kind: "episode" }, ({ episode }) => <EpisodeDetail episode={episode} {...detail} />)
        .with({ kind: "group" }, ({ group }) => <EpisodeGroupDetail group={group} {...detail} />)
        .exhaustive(),
    };
  };
  const groups = new Map(crashIncidents(episodes).map((group) => [incidentOf(group).id, group]));
  return {
    active,
    onActiveChange,
    items: [...groups.values()].map(toChartIncident),
    combine: (ids) =>
      toChartIncident(episodeGroup(ids.flatMap((id) => groups.get(id)?.episodes ?? []))),
  };
}
