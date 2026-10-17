const SENTINEL_KEY = "unkeyPreventLeaveSentinel";

export const SENTINEL_STATE = { [SENTINEL_KEY]: true };

export function isLeaveSentinel(state: unknown): boolean {
  return typeof state === "object" && state !== null && SENTINEL_KEY in state;
}

export function keepLeaveSentinel(state: unknown): typeof SENTINEL_STATE | null {
  return isLeaveSentinel(state) ? SENTINEL_STATE : null;
}

export function leavingLinkHref(event: MouseEvent): string | null {
  if (
    event.defaultPrevented ||
    event.button !== 0 ||
    event.metaKey ||
    event.ctrlKey ||
    event.shiftKey ||
    event.altKey
  ) {
    return null;
  }
  const anchor = event.target instanceof Element ? event.target.closest("a[href]") : null;
  if (!(anchor instanceof HTMLAnchorElement) || anchor.target === "_blank") {
    return null;
  }
  if (anchor.hasAttribute("download")) {
    return null;
  }
  const url = new URL(anchor.href, window.location.href);
  const samePage =
    url.origin === window.location.origin && url.pathname === window.location.pathname;
  return samePage ? null : anchor.href;
}

export type GuardMode = "off" | "confirm" | "hold";

// poppingSentinel: the guard went back over its own history entry, so the next popstate is ours.
export type LeavePhase = "idle" | "poppingSentinel" | "leaving";

export type LeaveEvent =
  | { type: "guardOn"; sentinelOnTop: boolean }
  | { type: "guardOff"; sentinelOnTop: boolean }
  | { type: "popstate"; sentinel: boolean; mode: GuardMode }
  | { type: "beforeunload"; dirty: boolean }
  | { type: "pagehide" }
  | { type: "leave" }
  | { type: "leaveBack" }
  | { type: "fallbackElapsed" }
  | { type: "keepEditing"; sentinelOnTop: boolean };

export type LeaveEffect =
  | "pushSentinel"
  | "historyBack"
  | "blockUnload"
  | "askBack"
  | "closePrompt"
  | "armFallback"
  | "disarmFallback"
  | "goToFallback";

export type LeaveStep = { phase: LeavePhase; effects: LeaveEffect[] };

export function nextLeaveStep(phase: LeavePhase, event: LeaveEvent): LeaveStep {
  switch (event.type) {
    case "guardOn":
      return event.sentinelOnTop
        ? { phase, effects: [] }
        : { phase: "idle", effects: ["pushSentinel"] };
    case "guardOff":
      return event.sentinelOnTop && phase !== "leaving"
        ? { phase: "poppingSentinel", effects: ["disarmFallback", "historyBack"] }
        : { phase: "idle", effects: ["disarmFallback"] };
    case "popstate":
      return popstateStep(phase, event);
    case "beforeunload":
      return { phase, effects: phase !== "leaving" && event.dirty ? ["blockUnload"] : [] };
    case "pagehide":
      return { phase: "idle", effects: ["disarmFallback"] };
    case "leave":
      return { phase: "leaving", effects: [] };
    case "leaveBack":
      return { phase: "leaving", effects: ["armFallback", "historyBack"] };
    case "fallbackElapsed":
      return { phase: "idle", effects: ["goToFallback"] };
    case "keepEditing":
      return {
        phase,
        effects: event.sentinelOnTop ? ["closePrompt"] : ["closePrompt", "pushSentinel"],
      };
  }
}

function popstateStep(
  phase: LeavePhase,
  { sentinel, mode }: { sentinel: boolean; mode: GuardMode },
): LeaveStep {
  switch (phase) {
    case "leaving":
      return { phase: "idle", effects: ["disarmFallback"] };
    case "poppingSentinel":
      return { phase: "idle", effects: ["pushSentinel"] };
    case "idle":
      return { phase, effects: [idleBackEffect(sentinel, mode)] };
  }
}

function idleBackEffect(sentinel: boolean, mode: GuardMode): LeaveEffect {
  if (sentinel) {
    return "closePrompt";
  }
  return mode === "hold" ? "pushSentinel" : "askBack";
}
