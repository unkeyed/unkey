import { useMemo } from "react";

const WINDOW_MS = 7 * 24 * 60 * 60 * 1000;
const MINUTE_MS = 60_000;

export function useOverviewWindow() {
  return useMemo(() => {
    const endTime = Math.floor(Date.now() / MINUTE_MS) * MINUTE_MS;
    return { startTime: endTime - WINDOW_MS, endTime };
  }, []);
}
