import { useCallback, useEffect, useState } from "react";

const TIMEOUT_MS = 60_000;

type Options<T> = {
  isReached: (target: T) => boolean;
  onSettled: () => void;
};

export type AwaitTarget<T> = {
  start: (target: T) => void;
  waiting: boolean;
};

// Promote, rollback, stop and wake return before the backend has written the
// state they cause, so a refetch right after the response reads the old row.
// start() records the state to wait for; the caller polls faster while
// `waiting` is true, and onSettled runs once isReached reports the state or a
// minute has passed.
export function useAwaitTarget<T>({ isReached, onSettled }: Options<T>): AwaitTarget<T> {
  const [target, setTarget] = useState<T | null>(null);
  const reached = target !== null && isReached(target);

  const settle = useCallback(() => {
    setTarget(null);
    onSettled();
  }, [onSettled]);

  useEffect(() => {
    if (target === null) {
      return;
    }
    if (reached) {
      settle();
      return;
    }
    const id = setTimeout(settle, TIMEOUT_MS);
    return () => clearTimeout(id);
  }, [target, reached, settle]);

  return { start: setTarget, waiting: target !== null };
}
