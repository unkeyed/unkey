"use client";

import { RESOURCE_LIST_VIEWS, type ResourceListView, parseResourceListView } from "@unkey/ui";
import { parseAsStringLiteral, useQueryState } from "nuqs";
import { useSyncExternalStore } from "react";

const DEFAULT_VIEW: ResourceListView = "grid";
const listeners = new Set<() => void>();

function subscribe(listener: () => void) {
  listeners.add(listener);
  window.addEventListener("storage", listener);
  return () => {
    listeners.delete(listener);
    window.removeEventListener("storage", listener);
  };
}

function readStoredView(storageKey: string): ResourceListView {
  try {
    return parseResourceListView(window.localStorage.getItem(storageKey)) ?? DEFAULT_VIEW;
  } catch {
    return DEFAULT_VIEW;
  }
}

function writeStoredView(storageKey: string, view: ResourceListView) {
  try {
    window.localStorage.setItem(storageKey, view);
  } catch {
    return;
  }
  for (const listener of listeners) {
    listener();
  }
}

export function useListView(
  storageKey: string,
): [ResourceListView, (view: ResourceListView) => void] {
  const [urlView, setUrlView] = useQueryState(
    "view",
    parseAsStringLiteral(RESOURCE_LIST_VIEWS).withOptions({ history: "replace" }),
  );
  const storedView = useSyncExternalStore(
    subscribe,
    () => readStoredView(storageKey),
    () => DEFAULT_VIEW,
  );

  const setView = (view: ResourceListView) => {
    writeStoredView(storageKey, view);
    setUrlView(view);
  };

  return [urlView ?? storedView, setView];
}
