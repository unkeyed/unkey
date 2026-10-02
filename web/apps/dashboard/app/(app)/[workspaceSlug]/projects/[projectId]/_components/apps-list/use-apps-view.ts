"use client";

import { parseAsStringLiteral, useQueryState } from "nuqs";
import { useSyncExternalStore } from "react";

export const APPS_VIEWS = ["grid", "list"] as const;
export type AppsView = (typeof APPS_VIEWS)[number];

const STORAGE_KEY = "apps-view";
const listeners = new Set<() => void>();

export function parseAppsView(value: unknown): AppsView | undefined {
  return APPS_VIEWS.find((view) => view === value);
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  window.addEventListener("storage", listener);
  return () => {
    listeners.delete(listener);
    window.removeEventListener("storage", listener);
  };
}

function readStoredView(): AppsView {
  try {
    return parseAppsView(window.localStorage.getItem(STORAGE_KEY)) ?? "grid";
  } catch {
    return "grid";
  }
}

function writeStoredView(view: AppsView) {
  try {
    window.localStorage.setItem(STORAGE_KEY, view);
  } catch {
    return;
  }
  for (const listener of listeners) {
    listener();
  }
}

export function useAppsView(): [AppsView, (view: AppsView) => void] {
  const [urlView, setUrlView] = useQueryState(
    "view",
    parseAsStringLiteral(APPS_VIEWS).withOptions({ history: "replace" }),
  );
  const storedView = useSyncExternalStore(subscribe, readStoredView, () => "grid" as const);

  const setView = (view: AppsView) => {
    writeStoredView(view);
    setUrlView(view);
  };

  return [urlView ?? storedView, setView];
}
