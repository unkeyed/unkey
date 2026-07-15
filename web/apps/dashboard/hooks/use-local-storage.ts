"use client";

import { type Dispatch, type SetStateAction, useCallback, useState } from "react";

// resolveValue handles lazy initial values with the same shape as React state initializers.
function resolveValue<TValue>(value: TValue | (() => TValue)): TValue {
  return value instanceof Function ? value() : value;
}

/**
 * useLocalStorage persists React state in localStorage using JSON serialization.
 *
 * The hook mirrors the subset of `usehooks-ts` behavior used by the dashboard:
 * it reads once during state initialization, accepts direct values or updater
 * functions, stores values with `JSON.stringify`, and returns a remove helper
 * that clears the key and resets state to the initial value.
 *
 * If localStorage is unavailable or contains invalid JSON, the hook returns the
 * initial value and keeps the UI usable.
 */
export function useLocalStorage<TValue>(
  key: string,
  initialValue: TValue | (() => TValue),
): readonly [TValue, Dispatch<SetStateAction<TValue>>, () => void] {
  const readValue = useCallback(() => {
    const resolvedInitialValue = resolveValue(initialValue);

    if (typeof window === "undefined") {
      return resolvedInitialValue;
    }

    try {
      const storedValue = window.localStorage.getItem(key);
      if (storedValue === null) {
        return resolvedInitialValue;
      }

      if (storedValue === "undefined") {
        return undefined as TValue;
      }

      return JSON.parse(storedValue) as TValue;
    } catch {
      return resolvedInitialValue;
    }
  }, [key, initialValue]);

  const [value, setValueState] = useState<TValue>(readValue);

  const setValue = useCallback(
    (nextValue: SetStateAction<TValue>) => {
      try {
        const resolvedValue = nextValue instanceof Function ? nextValue(readValue()) : nextValue;
        setValueState(resolvedValue);
        window.localStorage.setItem(key, JSON.stringify(resolvedValue));
      } catch {
        // Keep the UI usable when localStorage is blocked or unavailable.
      }
    },
    [key, readValue],
  );

  const removeValue = useCallback(() => {
    const resolvedInitialValue = resolveValue(initialValue);
    setValueState(resolvedInitialValue);
    try {
      window.localStorage.removeItem(key);
    } catch {
      // Keep the UI usable when localStorage is blocked or unavailable.
    }
  }, [key, initialValue]);

  return [value, setValue, removeValue] as const;
}
