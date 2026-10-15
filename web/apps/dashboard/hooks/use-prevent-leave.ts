"use client";

import type { DiscardChangesDialogProps } from "@unkey/ui";
import type { Route } from "next";
import { useRouter } from "next/navigation";
import { useEffect, useEffectEvent, useRef, useState } from "react";
import {
  type GuardMode,
  type LeaveEvent,
  type LeavePhase,
  SENTINEL_STATE,
  isLeaveSentinel,
  leavingLinkHref,
  nextLeaveStep,
} from "./prevent-leave-model";

// Neither popstate nor pagehide within this window means there is no entry to go back to.
const BACK_FALLBACK_DELAY_MS = 300;

type Leave = { kind: "back" } | { kind: "link"; href: string };

function sentinelOnTop(): boolean {
  return isLeaveSentinel(window.history.state);
}

function pushSentinel() {
  window.history.pushState(SENTINEL_STATE, "", window.location.href);
}

export function usePreventLeave(
  mode: GuardMode,
  dirty: boolean,
  fallbackHref: Route,
): {
  bypass: () => void;
  navigate: (href: Route) => void;
  prompt: DiscardChangesDialogProps;
} {
  const router = useRouter();
  const phaseRef = useRef<LeavePhase>("idle");
  const fallbackTimerRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const [pending, setPending] = useState<Leave | null>(null);

  const send = (event: LeaveEvent, nativeEvent?: Event) => {
    const step = nextLeaveStep(phaseRef.current, event);
    phaseRef.current = step.phase;
    for (const effect of step.effects) {
      switch (effect) {
        case "pushSentinel":
          pushSentinel();
          break;
        case "historyBack":
          window.history.back();
          break;
        case "blockUnload":
          nativeEvent?.preventDefault();
          break;
        case "askBack":
          setPending({ kind: "back" });
          break;
        case "closePrompt":
          setPending(null);
          break;
        case "armFallback":
          fallbackTimerRef.current = setTimeout(
            () => send({ type: "fallbackElapsed" }),
            BACK_FALLBACK_DELAY_MS,
          );
          break;
        case "disarmFallback":
          clearTimeout(fallbackTimerRef.current);
          break;
        case "goToFallback":
          router.replace(fallbackHref);
          break;
      }
    }
  };
  const sendFromListener = useEffectEvent(send);
  const popState = useEffectEvent((event: PopStateEvent) =>
    send({ type: "popstate", sentinel: isLeaveSentinel(event.state), mode }),
  );
  const beforeUnload = useEffectEvent((event: BeforeUnloadEvent) =>
    send({ type: "beforeunload", dirty }, event),
  );
  const leaveByLink = useEffectEvent((event: MouseEvent) => {
    const href = mode === "confirm" ? leavingLinkHref(event) : null;
    if (href === null) {
      return;
    }
    event.preventDefault();
    setPending({ kind: "link", href });
  });
  const enabled = mode !== "off";

  // biome-ignore lint/correctness/useExhaustiveDependencies: Biome 1.9 does not know useEffectEvent, whose result must stay out of the deps
  useEffect(() => {
    if (!enabled) {
      return;
    }
    const onBeforeUnload = (event: BeforeUnloadEvent) => beforeUnload(event);
    const onPopState = (event: PopStateEvent) => popState(event);
    const onPageHide = () => sendFromListener({ type: "pagehide" });
    const onClick = (event: MouseEvent) => leaveByLink(event);

    sendFromListener({ type: "guardOn", sentinelOnTop: sentinelOnTop() });
    window.addEventListener("beforeunload", onBeforeUnload);
    window.addEventListener("popstate", onPopState);
    window.addEventListener("pagehide", onPageHide);
    document.addEventListener("click", onClick, true);

    return () => {
      window.removeEventListener("beforeunload", onBeforeUnload);
      window.removeEventListener("popstate", onPopState);
      window.removeEventListener("pagehide", onPageHide);
      document.removeEventListener("click", onClick, true);
      sendFromListener({ type: "guardOff", sentinelOnTop: sentinelOnTop() });
    };
  }, [enabled]);

  const discard = (leave: Leave) => {
    setPending(null);
    if (leave.kind === "back") {
      send({ type: "leaveBack" });
      return;
    }
    send({ type: "leave" });
    const url = new URL(leave.href);
    if (url.origin !== window.location.origin) {
      window.location.assign(leave.href);
      return;
    }
    router.replace(`${url.pathname}${url.search}${url.hash}` as Route);
  };

  return {
    bypass: () => send({ type: "leave" }),
    navigate: (href) => {
      if (mode === "confirm") {
        setPending({ kind: "link", href: new URL(href, window.location.href).href });
      } else {
        router.push(href);
      }
    },
    prompt: {
      open: pending !== null,
      onKeepEditing: () => send({ type: "keepEditing", sentinelOnTop: sentinelOnTop() }),
      onDiscard: () => {
        if (pending) {
          discard(pending);
        }
      },
    },
  };
}
