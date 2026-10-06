import type { InstanceSummary } from "./instance-state";
import type { LogGroup, LogTone, Stage } from "./run-model";

export type TickerLine = { id: string; text: string; tone: LogTone; waiting?: boolean };

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
    return animate && state.shown
      ? { ...emptyTicker, shown: state.shown, leaving: true }
      : emptyTicker;
  }
  if (!state.shown || !animate) {
    return { ...emptyTicker, shown: line };
  }
  if (state.incoming) {
    return { ...state, queued: line };
  }
  return { ...state, incoming: line };
}

const RUNTIME_GROUP_ID = "runtime";

function lastLine(groups: LogGroup[]): TickerLine | null {
  const group = groups.at(-1);
  if (!group) {
    return null;
  }
  const line = group.lines.at(-1);
  return line
    ? { id: line.id, text: line.text, tone: line.tone }
    : { id: group.id, text: group.title, tone: group.failed ? "error" : "plain" };
}

const waitingLine = {
  building: {
    id: "building-waiting",
    text: "Preparing the build…",
    tone: "plain",
    waiting: true,
  },
  deploying: {
    id: "deploying-waiting",
    text: "Scheduling instances…",
    tone: "plain",
    waiting: true,
  },
} satisfies Record<string, TickerLine>;

export function stageTickerLine(
  stage: Stage,
  groups: LogGroup[],
  firstError: string | null,
  instances: InstanceSummary | null = null,
): TickerLine | null {
  if (stage.state === "failed") {
    const error = stage.error ?? firstError;
    return error ? { id: `${stage.key}-error`, text: error, tone: "error" } : null;
  }
  if (stage.state !== "active") {
    return null;
  }
  if (stage.key === "building") {
    return (
      lastLine(groups.filter((group) => group.id !== RUNTIME_GROUP_ID)) ?? waitingLine.building
    );
  }
  if (stage.key === "deploying") {
    return (
      lastLine(groups.filter((group) => group.id === RUNTIME_GROUP_ID)) ??
      (instances
        ? { id: `instances-${instances.text}`, text: instances.text, tone: instances.tone }
        : null) ??
      waitingLine.deploying
    );
  }
  return null;
}
