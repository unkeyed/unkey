import type { InstanceSummary } from "@/lib/collections/deploy/instance-status";
import type { Router } from "@/lib/trpc/routers";
import type { inferRouterOutputs } from "@trpc/server";
import type { BuildLogLine } from "../(deployment-progress)/build-logs/build-log-lines";
import { runningCountText } from "./deployment-health";
import type { Stage, StageKey } from "./stages-model";

type ContainerLogRow =
  inferRouterOutputs<Router>["deploy"]["deployment"]["runtimeLogs"]["logs"][number];

export type TickerTone = "plain" | "error" | "warn";

export type TickerLine = { id: string; text: string; tone: TickerTone; waiting?: boolean };

export type TickerSources = {
  buildLines: readonly BuildLogLine[];
  runtimeLogs: readonly ContainerLogRow[];
  instances: InstanceSummary | null;
};

type TickerState = {
  shown: TickerLine | null;
  incoming: TickerLine | null;
  queued: TickerLine | null;
  leaving: boolean;
};

export type TickerEvent =
  | { type: "receive"; line: TickerLine | null; animate: boolean }
  | { type: "settle" };

export const emptyTicker: TickerState = {
  shown: null,
  incoming: null,
  queued: null,
  leaving: false,
};

export function tickerReducer(state: TickerState, event: TickerEvent): TickerState {
  if (event.type === "settle") {
    if (state.leaving) {
      return emptyTicker;
    }
    if (!state.incoming) {
      return state;
    }
    const shown = state.incoming;
    const next = state.queued && state.queued.id !== shown.id ? state.queued : null;
    return { shown, incoming: next, queued: null, leaving: false };
  }
  const { line, animate } = event;
  if (state.leaving) {
    return line ? { ...emptyTicker, shown: line } : state;
  }
  const target = state.queued ?? state.incoming ?? state.shown;
  if (line?.id === target?.id) {
    return state;
  }
  if (!line) {
    if (!animate || !state.shown) {
      return emptyTicker;
    }
    return { ...emptyTicker, shown: state.incoming ?? state.shown, leaving: true };
  }
  if (!state.shown || !animate) {
    return { ...emptyTicker, shown: line };
  }
  if (state.incoming) {
    return { ...state, queued: line };
  }
  return { ...state, incoming: line };
}

const toneBySeverity: Partial<Record<string, TickerTone>> = {
  ERROR: "error",
  WARN: "warn",
};

const waitingLine = {
  build: { id: "build-waiting", text: "Preparing the build…", tone: "plain", waiting: true },
  instances: {
    id: "instances-waiting",
    text: "Scheduling instances…",
    tone: "plain",
    waiting: true,
  },
} satisfies Record<string, TickerLine>;

function lastText(text: string): string {
  return (
    text
      .split("\n")
      .map((part) => part.trim())
      .findLast((part) => part !== "") ?? ""
  );
}

function lastBuildLine(lines: readonly BuildLogLine[]): TickerLine | null {
  for (let index = lines.length - 1; index >= 0; index--) {
    const line = lines[index];
    if (line.kind !== "entry") {
      continue;
    }
    const text = lastText(line.text);
    if (line.tone === "error" && text !== "") {
      return { id: `${line.entry.stepId}-${line.entry.time}-${text}`, text, tone: "error" };
    }
    const step = line.entry.step.trim();
    if (step === "") {
      continue;
    }
    return text === "CACHED"
      ? { id: `${line.entry.stepId}-cached`, text: `${step} (cached)`, tone: "plain" }
      : { id: line.entry.stepId, text: step, tone: "plain" };
  }
  return null;
}

function lastRuntimeLine(logs: readonly ContainerLogRow[]): TickerLine | null {
  const latest = logs.reduce<ContainerLogRow | null>(
    (newest, log) => (!newest || log.time > newest.time ? log : newest),
    null,
  );
  if (!latest) {
    return null;
  }
  const text = lastText(latest.message);
  return {
    id: `${latest.time}-${latest.instance_id}-${text}`,
    text,
    tone: toneBySeverity[latest.severity.toUpperCase()] ?? "plain",
  };
}

function instancesLine(instances: InstanceSummary | null): TickerLine | null {
  if (!instances) {
    return null;
  }
  const text = runningCountText(instances);
  return {
    id: `instances-${text}`,
    text,
    tone: instances.state === "unhealthy" ? "error" : "plain",
  };
}

const liveLine: Record<StageKey, (sources: TickerSources) => TickerLine | null> = {
  build: (s) => lastBuildLine(s.buildLines) ?? waitingLine.build,
  instances: (s) =>
    lastRuntimeLine(s.runtimeLogs) ?? instancesLine(s.instances) ?? waitingLine.instances,
  domains: () => null,
};

export function stageTickerLine(
  { key, resolution }: Pick<Stage, "key" | "resolution">,
  sources: TickerSources,
): TickerLine | null {
  if (resolution.status === "error") {
    return { id: `${key}-error`, text: resolution.description, tone: "error" };
  }
  return resolution.status === "started" ? liveLine[key](sources) : null;
}
