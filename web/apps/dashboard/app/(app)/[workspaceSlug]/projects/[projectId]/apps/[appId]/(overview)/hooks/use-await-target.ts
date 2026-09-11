import { useCollectionPolling } from "@/lib/collections/use-collection-polling";
import { useCallback, useEffect, useRef, useState } from "react";

const POLL_INTERVAL_MS = 2_000;
const TIMEOUT_MS = 60_000;

type Options<T> = {
  isReached: (target: T) => boolean;
  poll: () => void;
  onSettled: () => void;
};

// Promote, rollback, stop and wake return before the backend has written the
// state they cause, so a single refetch after the response reads the old row.
// The returned function takes the state to wait for; the hook polls until
// isReached reports it (or a minute passes), then runs onSettled.
export function useAwaitTarget<T>({ isReached, poll, onSettled }: Options<T>): (target: T) => void {
  const [target, setTarget] = useState<T | null>(null);
  const pollRef = useRef(poll);
  pollRef.current = poll;

  const settle = useCallback(() => {
    setTarget(null);
    onSettled();
  }, [onSettled]);

  useCollectionPolling(poll, { intervalMs: POLL_INTERVAL_MS, enabled: target !== null });

  const reached = target !== null && isReached(target);

  useEffect(() => {
    if (reached) {
      settle();
    }
  }, [reached, settle]);

  useEffect(() => {
    if (target === null) {
      return;
    }
    const id = setTimeout(settle, TIMEOUT_MS);
    return () => clearTimeout(id);
  }, [target, settle]);

  return useCallback((next: T) => {
    setTarget(next);
    pollRef.current();
  }, []);
}
