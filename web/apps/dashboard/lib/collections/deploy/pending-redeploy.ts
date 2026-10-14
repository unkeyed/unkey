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

export function useSettingsBannerVisible(): boolean {
  return useSyncExternalStore(
    (cb) => saveStore.subscribe(cb),
    () => saveStore.savedCount > saveStore.dismissedAtCount,
  );
}

export function dismissSettingsBanner(): void {
  saveStore.dismiss();
}
