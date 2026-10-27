import type { V2EnvironmentsUpdateSettingsRequestBody } from "@unkey/api/models/components";
import { useSyncExternalStore } from "react";
import type { Deployment } from "./deployments";

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
  target: null as SettingsBannerTarget | null,
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
    showSettingsBanner();
    return result;
  });
}

type SettingsBannerTarget = {
  deployment: Pick<Deployment, "id" | "projectId" | "appId" | "environmentId">;
  environmentSlug: string;
};

export function showSettingsBanner(target: SettingsBannerTarget | null = null): void {
  saveStore.target = target;
  saveStore.savedCount++;
  saveStore.notify();
}

export function useSettingsBannerTarget(): SettingsBannerTarget | null {
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
