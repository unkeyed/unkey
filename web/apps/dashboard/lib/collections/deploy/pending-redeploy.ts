import type { V2EnvironmentsUpdateSettingsRequestBody } from "@unkey/api/models/components";
import { useSyncExternalStore } from "react";

export const NEXT_DEPLOY = "Changes apply to the next deployment";

/** Auto-deploy only decides whether a push starts a build, so it needs no redeploy. */
const SCOPE_ONLY: ReadonlySet<string> = new Set(["project", "app", "environment", "autoDeploy"]);

export function appliesOnNextDeploy(body: V2EnvironmentsUpdateSettingsRequestBody): boolean {
  return Object.keys(body).some((key) => !SCOPE_ONLY.has(key));
}

/**
 * Counts saves that the running deployment does not have yet. Every collection
 * whose writes apply on the next deploy reports here, so the pending-redeploy
 * banner reacts to all of them.
 */
const saveStore = {
  savedCount: 0,
  dismissedAtCount: 0,
  listeners: new Set<() => void>(),
  notify() {
    for (const cb of this.listeners) {
      cb();
    }
  },
  subscribe(cb: () => void): () => void {
    this.listeners.add(cb);
    return () => {
      this.listeners.delete(cb);
    };
  },
  dismiss() {
    this.dismissedAtCount = this.savedCount;
    this.notify();
  },
};

export function trackSave<T>(promise: Promise<T>): Promise<T> {
  return promise.then((result) => {
    saveStore.savedCount++;
    saveStore.notify();
    return result;
  });
}

/** Returns true when there are saves the user hasn't dismissed yet. Survives navigation. */
export function useSettingsBannerVisible(): boolean {
  return useSyncExternalStore(
    (cb) => saveStore.subscribe(cb),
    () => saveStore.savedCount > saveStore.dismissedAtCount,
  );
}

/** Dismisses the pending-redeploy banner until a new save occurs. */
export function dismissSettingsBanner(): void {
  saveStore.dismiss();
}
