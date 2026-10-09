"use client";

import type { DiscardChangesDialogProps } from "@unkey/ui";
import type { Route } from "next";
import { useRouter } from "next/navigation";
import { createContext, use, useCallback, useEffect, useRef, useState } from "react";

const SENTINEL_KEY = "unkeyPreventLeaveSentinel";

// Neither popstate nor pagehide within this window means there is no entry to go back to.
const BACK_FALLBACK_DELAY_MS = 300;

type Leave = { kind: "back" } | { kind: "link"; href: string };

export function isLeaveSentinel(state: unknown): boolean {
  return typeof state === "object" && state !== null && SENTINEL_KEY in state;
}

function pushSentinel() {
  window.history.pushState({ [SENTINEL_KEY]: true }, "", window.location.href);
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

export function usePreventLeave(
  enabled: boolean,
  fallbackHref: Route,
): {
  bypass: () => void;
  navigate: (href: Route) => void;
  leavePrompt: DiscardChangesDialogProps;
} {
  const router = useRouter();
  const skipNextRef = useRef(false);
  const leavingRef = useRef(false);
  const poppingSentinelRef = useRef(false);
  const fallbackTimerRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const [leave, setLeave] = useState<Leave | null>(null);

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
      if (poppingSentinelRef.current) {
        poppingSentinelRef.current = false;
        pushSentinel();
        return;
      }
      if (skipNextRef.current) {
        return;
      }
      if (isLeaveSentinel(event.state)) {
        setLeave(null);
        return;
      }
      setLeave({ kind: "back" });
    };

    const handleClick = (event: MouseEvent) => {
      const href = leavingLinkHref(event);
      if (href === null) {
        return;
      }
      event.preventDefault();
      setLeave({ kind: "link", href });
    };

    if (!isLeaveSentinel(window.history.state)) {
      poppingSentinelRef.current = false;
      pushSentinel();
    }
    window.addEventListener("beforeunload", handleBeforeUnload);
    window.addEventListener("popstate", handlePopState);
    window.addEventListener("pagehide", stopLeaving);
    document.addEventListener("click", handleClick, true);

    return () => {
      const leaving = leavingRef.current;
      stopLeaving();
      window.removeEventListener("beforeunload", handleBeforeUnload);
      window.removeEventListener("popstate", handlePopState);
      window.removeEventListener("pagehide", stopLeaving);
      document.removeEventListener("click", handleClick, true);
      if (isLeaveSentinel(window.history.state) && !leaving) {
        poppingSentinelRef.current = true;
        window.history.back();
      }
    };
  }, [enabled]);

  const discard = (pending: Leave) => {
    leavingRef.current = true;
    if (pending.kind === "link") {
      const url = new URL(pending.href);
      if (url.origin !== window.location.origin) {
        window.location.assign(pending.href);
        return;
      }
      router.replace(`${url.pathname}${url.search}${url.hash}` as Route);
      return;
    }
    fallbackTimerRef.current = setTimeout(() => {
      leavingRef.current = false;
      router.replace(fallbackHref);
    }, BACK_FALLBACK_DELAY_MS);
    window.history.back();
  };

  const navigate = useCallback(
    (href: Route) => {
      if (enabled) {
        setLeave({ kind: "link", href: new URL(href, window.location.href).href });
      } else {
        router.push(href);
      }
    },
    [enabled, router],
  );

  return {
    bypass,
    navigate,
    leavePrompt: {
      open: leave !== null,
      onKeepEditing: () => {
        setLeave(null);
        if (!isLeaveSentinel(window.history.state)) {
          pushSentinel();
        }
      },
      onDiscard: () => {
        if (leave) {
          setLeave(null);
          discard(leave);
        }
      },
    },
  };
}

export const GuardedNavigateContext = createContext<((href: Route) => void) | null>(null);

export function useGuardedNavigate(): (href: Route) => void {
  const navigate = use(GuardedNavigateContext);
  const router = useRouter();
  return navigate ?? ((href) => router.push(href));
}
