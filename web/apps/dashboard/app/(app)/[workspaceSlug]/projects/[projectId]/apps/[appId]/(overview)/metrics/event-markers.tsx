"use client";

import type { ChartMarkers } from "@/components/charts/chart-markers";
import type { Route } from "next";
import { EpisodeDetail } from "./event-detail";
import { type CrashEpisode, episodeMarkers } from "./events-model";
import type { LogsTarget } from "./instances-model";

export function withEpisodeMarkers(
  markers: ChartMarkers,
  episodes: CrashEpisode[],
  detail: { logsHref: (target: LogsTarget) => Route; withDate: boolean },
): ChartMarkers {
  const byId = new Map(episodes.map((episode) => [`episode:${episode.id}`, episode]));
  return {
    ...markers,
    items: [...markers.items, ...episodeMarkers(episodes)],
    renderPopover: (id) => {
      const episode = byId.get(id);
      return episode ? <EpisodeDetail episode={episode} {...detail} /> : markers.renderPopover(id);
    },
  };
}
