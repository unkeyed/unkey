import * as React from "react";

/**
 * Local replacement for Radix's useControllableState (dependency removed in the Base UI migration).
 *
 * Base UI does not ship a public `useControllableState`, so we replicate the
 * Radix API surface the codebase relies on:
 *
 *   const [state, setState] = useControllableState({ prop, defaultProp, onChange })
 *
 * Semantics:
 * - When `prop` is defined the value is controlled: `prop` always wins and
 *   internal state is never used to derive the value.
 * - When `prop` is undefined the value is uncontrolled: internal state is
 *   seeded from `defaultProp`.
 * - `setState` accepts a value or an updater function. It calls `onChange`
 *   with the next value (only when it actually changes) and updates the
 *   internal state only in the uncontrolled case.
 */
export type UseControllableStateParams<T> = {
  /** The controlled value. When defined, the hook is controlled. */
  prop?: T | undefined;
  /** The initial value used while uncontrolled. */
  defaultProp: T;
  /** Called with the next value whenever `setState` produces a change. */
  onChange?: (value: T) => void;
  /**
   * Accepted for API-compatibility with the Radix hook. Only used for
   * debugging warnings there; intentionally unused here.
   */
  caller?: string;
};

type SetStateFn<T> = (prevState: T) => T;

export function useControllableState<T>({
  prop,
  defaultProp,
  onChange,
}: UseControllableStateParams<T>): [T, React.Dispatch<React.SetStateAction<T>>] {
  const [uncontrolledState, setUncontrolledState] = React.useState<T>(defaultProp);

  const isControlled = prop !== undefined;
  const value = (isControlled ? prop : uncontrolledState) as T;

  // Keep the latest onChange without re-creating the setter on every render.
  const onChangeRef = React.useRef(onChange);
  React.useEffect(() => {
    onChangeRef.current = onChange;
  });

  const setValue = React.useCallback<React.Dispatch<React.SetStateAction<T>>>(
    (next) => {
      if (isControlled) {
        const nextValue = typeof next === "function" ? (next as SetStateFn<T>)(prop as T) : next;
        if (!Object.is(nextValue, prop)) {
          onChangeRef.current?.(nextValue);
        }
      } else {
        setUncontrolledState((prevState) => {
          const nextValue = typeof next === "function" ? (next as SetStateFn<T>)(prevState) : next;
          if (!Object.is(nextValue, prevState)) {
            onChangeRef.current?.(nextValue);
          }
          return nextValue;
        });
      }
    },
    [isControlled, prop],
  );

  return [value, setValue];
}
