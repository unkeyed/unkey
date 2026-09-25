"use client";

import { parseAsStringLiteral, useQueryState } from "nuqs";
import { useEffect, useState } from "react";
import { APPS_VIEWS, type AppsView } from "./apps-list-controls";

const STORAGE_KEY = "apps-view";

function parseView(value: string | null): AppsView | undefined {
  return APPS_VIEWS.find((view) => view === value);
}

export function useAppsView(): [AppsView, (view: AppsView) => void] {
  const [urlView, setUrlView] = useQueryState(
    "view",
    parseAsStringLiteral(APPS_VIEWS).withOptions({ history: "replace" }),
  );
  const [storedView, setStoredView] = useState<AppsView>("grid");

  useEffect(() => {
    try {
      const stored = parseView(window.localStorage.getItem(STORAGE_KEY));
      if (stored) {
        setStoredView(stored);
      }
    } catch {}
  }, []);

  const setView = (view: AppsView) => {
    setStoredView(view);
    setUrlView(view);
    try {
      window.localStorage.setItem(STORAGE_KEY, view);
    } catch {}
  };

  return [urlView ?? storedView, setView];
}
