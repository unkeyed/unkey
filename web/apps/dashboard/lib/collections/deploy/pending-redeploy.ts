import type { V2EnvironmentsUpdateSettingsRequestBody } from "@unkey/api/models/components";
import { useSyncExternalStore } from "react";

export const NEXT_DEPLOY = "Changes apply to the next deployment";

const APPLIES_WITHOUT_REDEPLOY: ReadonlySet<string> = new Set([
  "project",
  "app",
  "environment",
  "autoDeploy",
] satisfies ReadonlyArray<keyof V2EnvironmentsUpdateSettingsRequestBody>);

export function appliesOnNextDeploy(body: V2EnvironmentsUpdateSettingsRequestBody): boolean {
  return Object.keys(body).some((key) => !APPLIES_WITHOUT_REDEPLOY.has(key));
}

const saveStore = {
  savedCount: 0,
  dismissedAtCount: 0,
  target: null as RedeployScope | null,
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

export function trackSave<T>(promise: Promise<T>, scope: RedeployScope): Promise<T> {
  return promise.then((result) => {
    showSettingsBanner(scope);
    return result;
  });
}

// An app-wide change applies to production. Each save passes a fresh object, so
// the timer of an earlier save cannot dismiss a later one.
export type RedeployScope = { kind: "app" } | { kind: "environments"; environmentIds: string[] };

export function showSettingsBanner(scope: RedeployScope): void {
  saveStore.target = scope;
  saveStore.savedCount++;
  saveStore.notify();
}

export function useSettingsBannerTarget(): RedeployScope | null {
  return useSyncExternalStore(
    (cb) => saveStore.subscribe(cb),
    () => saveStore.target,
  );
}

export function useSettingsBannerVisible(): boolean {
  return useSyncExternalStore(
    (cb) => saveStore.subscribe(cb),
    () => saveStore.savedCount > saveStore.dismissedAtCount,
  );
}

export function dismissSettingsBanner(target = saveStore.target): void {
  if (target === saveStore.target) {
    saveStore.dismiss();
  }
}
