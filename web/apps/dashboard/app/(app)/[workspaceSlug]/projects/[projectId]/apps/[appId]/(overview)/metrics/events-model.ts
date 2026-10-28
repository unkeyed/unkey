import type { ChartMarker } from "@/components/charts/chart-markers";
import { toInstanceKey } from "@/lib/collections/deploy/instance-status";
import type { InstanceEvent, InstanceEventKind } from "@/lib/metrics/app-metrics";
import type { FilterSelection } from "./instances-model";

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
export const EPISODE_GAP_MS = 10 * 60_000;

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
  latestMs: number;
  restarts: number;
  exitCode: number | null;
  reason: string;
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
  const exit = problems.findLast((e) => e.exitCode !== null) ?? last;
  return {
    id: `${toInstanceKey(first.instance, first.region)}:${first.atMs}`,
    instance: first.instance,
    instanceId: last.instanceId,
    region: first.region,
    deploymentId: last.deploymentId,
    kind: EPISODE_KIND_ORDER.find((kind) => problems.some((e) => e.kind === kind)) ?? first.kind,
    startMs: first.atMs,
    endMs: recoveredMs ?? (ongoing ? gridEndMs : last.atMs),
    latestMs: last.atMs,
    restarts: Math.max(problems.filter((e) => EVENT_KIND[e.kind].restart).length, 1),
    exitCode: exit.exitCode,
    reason: exit.reason,
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

export function logsWindow(episode: CrashEpisode): { startMs: number; endMs: number } {
  return { startMs: episode.startMs - LOGS_LEAD_MS, endMs: episode.endMs + LOGS_TAIL_MS };
}

export function episodeLabel(episode: CrashEpisode): string {
  const { label } = EVENT_KIND[episode.kind];
  return episode.restarts > 1 ? `${label}, ${episode.restarts} restarts` : label;
}

export function episodeMarkers(episodes: ReadonlyArray<CrashEpisode>): ChartMarker[] {
  return episodes.map((episode) => ({
    id: `episode:${episode.id}`,
    atMs: episode.startMs,
    span: { endMs: episode.endMs },
    shape: "diamond",
    label: episodeLabel(episode),
  }));
}
