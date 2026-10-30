import { formatSpan } from "@/components/charts/format-interval";
import { summarizeExit } from "@/lib/collections/deploy/exit-reasons";
import { isFatalWaitingReason, toInstanceKey } from "@/lib/collections/deploy/instance-status";
import { plural } from "@/lib/fmt";
import type { InstanceEvent, InstanceEventKind } from "@/lib/metrics/app-metrics";
import type { FilterSelection, LogsTarget } from "./instances-model";

type EventTone = "error" | "warning" | "neutral";

type KindCopy = { label: string; tone: EventTone; restart: boolean };

export const EVENT_KIND: Record<InstanceEventKind, KindCopy> = {
  crashed: {
    label: "Crashed",
    tone: "error",
    restart: true,
  },
  oom: {
    label: "Out of memory",
    tone: "error",
    restart: true,
  },
  crashloop: {
    label: "Crash loop",
    tone: "error",
    restart: false,
  },
  "image-pull": {
    label: "Image pull",
    tone: "warning",
    restart: false,
  },
  config: {
    label: "Config error",
    tone: "error",
    restart: false,
  },
  completed: {
    label: "Exited",
    tone: "neutral",
    restart: false,
  },
  started: {
    label: "Started",
    tone: "neutral",
    restart: false,
  },
};

function isProblem(event: InstanceEvent): boolean {
  return EVENT_KIND[event.kind].tone !== "neutral";
}

export function eventsInScope(
  events: ReadonlyArray<InstanceEvent>,
  selection: Pick<FilterSelection, "regions" | "instances">,
): InstanceEvent[] {
  return events.filter(
    (e) =>
      (selection.regions.length === 0 || selection.regions.includes(e.region)) &&
      (selection.instances.length === 0 || selection.instances.includes(e.instance)),
  );
}

// Kubernetes caps the crash-loop restart backoff at 5 minutes, so a loop never goes quiet this long.
const EPISODE_GAP_MS = 10 * 60_000;

const EPISODE_KIND_ORDER: InstanceEventKind[] = [
  "oom",
  "crashed",
  "crashloop",
  "config",
  "image-pull",
];

export type CrashEpisode = {
  id: string;
  instance: string;
  instanceId: string | null;
  region: string;
  deploymentId: string;
  kind: InstanceEventKind;
  startMs: number;
  endMs: number;
  ongoing: boolean;
  latestMs: number;
  restarts: number;
  exitCode: number | null;
  reason: string;
  message: string | null;
};

type Run = { problems: InstanceEvent[]; recoveredMs: number | null };

function instanceEpisodes(events: InstanceEvent[], gridEndMs: number): CrashEpisode[] {
  const runs: Run[] = [];
  for (const event of events.toSorted((a, b) => a.atMs - b.atMs)) {
    const run = runs.at(-1);
    const last = run?.problems.at(-1);
    const within = last !== undefined && event.atMs - last.atMs <= EPISODE_GAP_MS;
    if (event.kind === "started") {
      if (run && within) {
        run.recoveredMs ??= event.atMs;
      }
    } else if (run && within) {
      run.problems.push(event);
      run.recoveredMs = null;
    } else {
      runs.push({ problems: [event], recoveredMs: null });
    }
  }
  return runs.map(({ problems, recoveredMs }) => episodeOf(problems, recoveredMs, gridEndMs));
}

function episodeOf(
  problems: InstanceEvent[],
  recoveredMs: number | null,
  gridEndMs: number,
): CrashEpisode {
  const [first] = problems;
  const last = problems[problems.length - 1];
  const ongoing = recoveredMs === null && gridEndMs - last.atMs <= EPISODE_GAP_MS;
  const terminated = problems.findLast((e) => e.exitCode !== null);
  return {
    id: `${toInstanceKey(first.instance, first.region)}:${first.atMs}`,
    instance: first.instance,
    instanceId: last.instanceId,
    region: first.region,
    deploymentId: last.deploymentId,
    kind: EPISODE_KIND_ORDER.find((kind) => problems.some((e) => e.kind === kind)) ?? first.kind,
    startMs: first.atMs,
    endMs: recoveredMs ?? (ongoing ? gridEndMs : last.atMs),
    ongoing,
    latestMs: last.atMs,
    restarts: Math.max(problems.filter((e) => EVENT_KIND[e.kind].restart).length, 1),
    exitCode: terminated?.exitCode ?? null,
    reason: (terminated ?? last).reason,
    message: terminated?.message ?? null,
  };
}

// Kubernetes restarts a crashed container within seconds; a pod that stops and never
// comes back was shut down (a rollout or scale-down sends SIGTERM, exit 143), not crashed.
const RESTART_GRACE_MS = 2 * 60_000;

function withoutShutdowns(events: InstanceEvent[], gridEndMs: number): InstanceEvent[] {
  const sorted = events.toSorted((a, b) => a.atMs - b.atMs);
  return sorted.filter(
    (e, i) =>
      e.kind !== "crashed" || i < sorted.length - 1 || gridEndMs - e.atMs <= RESTART_GRACE_MS,
  );
}

export function crashEpisodes(
  events: ReadonlyArray<InstanceEvent>,
  gridEndMs: number,
): CrashEpisode[] {
  const relevant = events.filter((e) => isProblem(e) || e.kind === "started");
  return [...Map.groupBy(relevant, (e) => toInstanceKey(e.instance, e.region)).values()]
    .flatMap((events) => instanceEpisodes(withoutShutdowns(events, gridEndMs), gridEndMs))
    .sort((a, b) => a.startMs - b.startMs || a.instance.localeCompare(b.instance));
}

const LOGS_LEAD_MS = 15 * 60_000;
const LOGS_TAIL_MS = 5 * 60_000;

export type Span = { startMs: number; endMs: number };

export function logsWindow(span: Span): Span {
  return { startMs: span.startMs - LOGS_LEAD_MS, endMs: span.endMs + LOGS_TAIL_MS };
}

export type ErrorGroup = {
  title: string;
  detail: string | null;
  count: number;
  kinds: InstanceEventKind[];
  regions: string[];
  logs: LogsTarget;
};

export type EpisodeGroup = {
  span: Span;
  episodes: CrashEpisode[];
  errors: ErrorGroup[];
  logs: LogsTarget;
};

function spanOf(episodes: ReadonlyArray<CrashEpisode>): Span {
  return {
    startMs: Math.min(...episodes.map((e) => e.startMs)),
    endMs: Math.max(...episodes.map((e) => e.endMs)),
  };
}

export function spanEnd(episodes: ReadonlyArray<CrashEpisode>): number | "now" {
  return episodes.some((e) => e.ongoing) ? "now" : spanOf(episodes).endMs;
}

function logsInstanceIds(episodes: ReadonlyArray<CrashEpisode>): string[] {
  const ids = episodes.map((e) => e.instanceId);
  return ids.every((id) => id !== null) ? [...new Set(ids)] : [];
}

export function episodeLogs(episodes: ReadonlyArray<CrashEpisode>): LogsTarget {
  return {
    deploymentIds: [...new Set(episodes.map((e) => e.deploymentId))],
    instanceIds: logsInstanceIds(episodes),
    window: logsWindow(spanOf(episodes)),
  };
}

type EpisodeError = {
  title: string;
  detail: string | null;
  exit: { code: number; reason: string } | null;
};

function exitTitle({ exitCode, reason }: CrashEpisode): string {
  const { label, exit } = summarizeExit({
    exitCode,
    reason,
    signal: null,
    statusReason: isFatalWaitingReason(reason) ? reason : null,
  });
  return [label, exit].filter(Boolean).join(" · ");
}

export function episodeError(episode: CrashEpisode): EpisodeError {
  if (episode.message === null) {
    return { title: exitTitle(episode), detail: null, exit: null };
  }
  const [title, ...rest] = episode.message.trim().split("\n");
  return {
    title: title.trim(),
    detail: rest.join("\n").trim() || null,
    exit: episode.exitCode === null ? null : { code: episode.exitCode, reason: episode.reason },
  };
}

function errorsOf(episodes: CrashEpisode[]): ErrorGroup[] {
  const byKey = Map.groupBy(
    episodes.map((episode) => ({ episode, ...episodeError(episode) })),
    (e) => e.title,
  );
  return [...byKey]
    .map(([title, members]) => {
      const matching = members.map((m) => m.episode);
      return {
        title,
        detail: members[0].detail,
        count: matching.length,
        kinds: EPISODE_KIND_ORDER.filter((kind) => matching.some((e) => e.kind === kind)),
        regions: [...new Set(matching.map((e) => e.region))],
        logs: episodeLogs(matching),
      };
    })
    .toSorted((a, b) => b.count - a.count);
}

export function episodeGroup(group: ReadonlyArray<CrashEpisode>): EpisodeGroup {
  const episodes = group.toSorted(
    (a, b) => a.startMs - b.startMs || a.instance.localeCompare(b.instance),
  );
  return {
    span: spanOf(episodes),
    episodes,
    errors: errorsOf(episodes),
    logs: episodeLogs(episodes),
  };
}

const INCIDENT_GAP_MS = 15 * 60_000;

export function crashIncidents(episodes: ReadonlyArray<CrashEpisode>): EpisodeGroup[] {
  const runs: { episodes: CrashEpisode[]; endMs: number }[] = [];
  for (const episode of episodes.toSorted((a, b) => a.startMs - b.startMs)) {
    const run = runs.at(-1);
    if (run && episode.startMs - run.endMs <= INCIDENT_GAP_MS) {
      run.episodes.push(episode);
      run.endMs = Math.max(run.endMs, episode.endMs);
    } else {
      runs.push({ episodes: [episode], endMs: episode.endMs });
    }
  }
  return runs.map((run) => episodeGroup(run.episodes));
}

export function episodeLabel(episode: CrashEpisode): string {
  const { label } = EVENT_KIND[episode.kind];
  return episode.restarts > 1 ? `${label}, ${episode.restarts} restarts` : label;
}

type CrashIncident = { id: string; span: Span; label: string } & (
  | { kind: "episode"; episode: CrashEpisode }
  | { kind: "group"; group: EpisodeGroup }
);

export function incidentOf(group: EpisodeGroup): CrashIncident {
  const [first] = group.episodes;
  const base = { id: `incident:${first.id}`, span: group.span };
  const range = formatSpan(group.span.startMs, spanEnd(group.episodes), true);
  return group.episodes.length === 1
    ? { ...base, kind: "episode", episode: first, label: `${episodeLabel(first)}, ${range}` }
    : {
        ...base,
        kind: "group",
        group,
        label: `${plural(group.episodes.length, "crash event")}, ${range}`,
      };
}
