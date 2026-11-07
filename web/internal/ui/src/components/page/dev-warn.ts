import { useEffect } from "react";

export const isDev = typeof process !== "undefined" && process.env.NODE_ENV !== "production";

export function useWarnIfMissing(present: boolean, message: string) {
  useEffect(() => {
    if (!isDev || present) {
      return;
    }
    const timer = setTimeout(() => console.warn(message), 0);
    return () => clearTimeout(timer);
  }, [present, message]);
}
