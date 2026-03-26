"use client";

import { useCallback, useEffect, useRef } from "react";

const CONFIRM_MESSAGE = "Changes you made may not be saved.";

/**
 * Prevents the user from accidentally leaving the page via tab close, refresh,
 * or browser back navigation. When `enabled` is true the hook:
 *
 * - Listens for `beforeunload` to intercept tab close / refresh.
 * - Pushes a sentinel history entry and listens for `popstate` to intercept the
 *   browser back button, showing a `window.confirm` dialog.
 *
 * Returns a `bypass` function that can be called before an intentional
 * navigation (e.g. an OAuth redirect) to skip the confirmation.
 */
export function usePreventLeave(enabled = true): { bypass: () => void } {
  const skipNextRef = useRef(false);

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

    const handleBeforeUnload = (e: BeforeUnloadEvent) => {
      if (skipNextRef.current) {
        return;
      }
      e.preventDefault();
    };

    // Push a sentinel history entry so pressing back triggers popstate instead
    // of leaving the page. On popstate, show a confirm dialog — if the user
    // cancels we re-push the entry; if they confirm we actually navigate back.
    window.history.pushState(null, "", window.location.href);

    const handlePopState = () => {
      if (skipNextRef.current) {
        return;
      }
      if (window.confirm(CONFIRM_MESSAGE)) {
        window.history.back();
      } else {
        window.history.pushState(null, "", window.location.href);
      }
    };

    window.addEventListener("beforeunload", handleBeforeUnload);
    window.addEventListener("popstate", handlePopState);

    return () => {
      window.removeEventListener("beforeunload", handleBeforeUnload);
      window.removeEventListener("popstate", handlePopState);
    };
  }, [enabled]);

  return { bypass };
}
