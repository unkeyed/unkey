import { useSyncExternalStore } from "react";

const subscribeNever = () => () => {};

export function useHydrated(): boolean {
  return useSyncExternalStore(
    subscribeNever,
    () => true,
    () => false,
  );
}
