import type { Loadable } from "./loadable";

export const loading = { status: "loading" } as const satisfies Loadable<never>;

export function ready<T>(data: T): Extract<Loadable<T>, { status: "ready" }> {
  return { status: "ready", data, stale: null };
}

export function failed(): Extract<Loadable<never>, { status: "failed" }> {
  return { status: "failed", error: new Error("boom"), retry: () => {} };
}
