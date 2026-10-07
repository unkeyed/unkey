"use client";

import type { DiscardChangesDialogProps } from "@/components/discard-changes-dialog";
import type { Route } from "next";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useRef, useState } from "react";

const SENTINEL_KEY = "unkeyPreventLeaveSentinel";

// history.back() fires popstate or pagehide within a frame or two. If neither
// arrives, there is no entry to go back to (the tab opened on this page).
const BACK_FALLBACK_DELAY_MS = 300;

// React runs effects twice in development; without the check a second sentinel
// would make "Discard changes" land on the first one instead of leaving.
export function isLeaveSentinel(state: unknown): boolean {
  return typeof state === "object" && state !== null && SENTINEL_KEY in state;
}

function pushSentinel() {
  window.history.pushState({ [SENTINEL_KEY]: true }, "", window.location.href);
}

/**
 * Prevents the user from accidentally leaving the page via tab close, refresh,
 * or browser back navigation. When `enabled` is true the hook:
 *
 * - Listens for `beforeunload` to intercept tab close / refresh.
 * - Pushes a sentinel history entry and listens for `popstate` to intercept the
 *   browser back button, opening the dialog described by `backPrompt`.
 *
 * Discarding goes back one entry, or to `fallbackHref` when there is none.
 *
 * Returns a `bypass` function that can be called before an intentional
 * navigation (e.g. an OAuth redirect) to skip the confirmation.
 */
export function usePreventLeave(
  enabled: boolean,
  fallbackHref: Route,
): {
  bypass: () => void;
  backPrompt: DiscardChangesDialogProps;
} {
  const router = useRouter();
  const skipNextRef = useRef(false);
  const leavingRef = useRef(false);
  const fallbackTimerRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const [confirmingBack, setConfirmingBack] = useState(false);

  // Allows exactly the next beforeunload event through (e.g. an OAuth redirect)
  // without disabling the guard permanently. The flag auto-resets on the next
  // tick so the guard stays active for everything else.
  const bypass = useCallback(() => {
    skipNextRef.current = true;
    setTimeout(() => {
      skipNextRef.current = false;
    }, 0);
  }, []);

  useEffect(() => {
    if (!enabled) {
      return;
    }

    const stopLeaving = () => {
      clearTimeout(fallbackTimerRef.current);
      leavingRef.current = false;
    };

    const handleBeforeUnload = (e: BeforeUnloadEvent) => {
      if (skipNextRef.current || leavingRef.current) {
        return;
      }
      e.preventDefault();
    };

    const handlePopState = (event: PopStateEvent) => {
      if (leavingRef.current) {
        stopLeaving();
        return;
      }
      if (skipNextRef.current) {
        return;
      }
      if (isLeaveSentinel(event.state)) {
        setConfirmingBack(false);
        return;
      }
      setConfirmingBack(true);
    };

    if (!isLeaveSentinel(window.history.state)) {
      pushSentinel();
    }
    window.addEventListener("beforeunload", handleBeforeUnload);
    window.addEventListener("popstate", handlePopState);
    window.addEventListener("pagehide", stopLeaving);

    return () => {
      stopLeaving();
      window.removeEventListener("beforeunload", handleBeforeUnload);
      window.removeEventListener("popstate", handlePopState);
      window.removeEventListener("pagehide", stopLeaving);
    };
  }, [enabled]);

  return {
    bypass,
    backPrompt: {
      open: confirmingBack,
      onKeepEditing: () => {
        setConfirmingBack(false);
        if (!isLeaveSentinel(window.history.state)) {
          pushSentinel();
        }
      },
      onDiscard: () => {
        setConfirmingBack(false);
        leavingRef.current = true;
        fallbackTimerRef.current = setTimeout(() => {
          leavingRef.current = false;
          router.replace(fallbackHref);
        }, BACK_FALLBACK_DELAY_MS);
        window.history.back();
      },
    },
  };
}
