"use client";

import { useCallback, useState } from "react";

const STORAGE_KEY = "unkey-sidebar-usage-minimised";

function read(): boolean {
  try {
    return window.localStorage.getItem(STORAGE_KEY) === "true";
  } catch {
    return false;
  }
}

// Reading storage in the initialiser is only safe because the sidebar first
// renders after hydration — `(app)/layout.tsx` gates on the workspace query.
export function useMinimised(): [boolean, (next: boolean) => void] {
  const [minimised, setState] = useState(read);

  const setMinimised = useCallback((next: boolean) => {
    setState(next);
    try {
      window.localStorage.setItem(STORAGE_KEY, String(next));
    } catch {}
  }, []);

  return [minimised, setMinimised];
}
