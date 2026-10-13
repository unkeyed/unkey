const SENTINEL_KEY = "unkeyPreventLeaveSentinel";

export const SENTINEL_STATE = { [SENTINEL_KEY]: true };

export function isLeaveSentinel(state: unknown): boolean {
  return typeof state === "object" && state !== null && SENTINEL_KEY in state;
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

// poppingSentinel: the guard went back over its own history entry, so the next popstate is ours.
export type LeavePhase = "idle" | "poppingSentinel" | "leaving";

export type LeaveEvent =
  | { type: "guardOn"; sentinelOnTop: boolean }
  | { type: "guardOff"; sentinelOnTop: boolean }
  | { type: "popstate"; sentinel: boolean }
  | { type: "beforeunload" }
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
      return popstateStep(phase, event.sentinel);
    case "beforeunload":
      return { phase, effects: phase === "leaving" ? [] : ["blockUnload"] };
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

function popstateStep(phase: LeavePhase, sentinel: boolean): LeaveStep {
  switch (phase) {
    case "leaving":
      return { phase: "idle", effects: ["disarmFallback"] };
    case "poppingSentinel":
      return { phase: "idle", effects: ["pushSentinel"] };
    case "idle":
      return { phase, effects: [sentinel ? "closePrompt" : "askBack"] };
  }
}
