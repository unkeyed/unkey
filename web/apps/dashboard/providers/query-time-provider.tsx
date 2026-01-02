"use client";

import { type ReactNode, createContext, useContext, useEffect, useState } from "react";

let queryTimestamp = Date.now();
let subscribers: ((timestamp: number) => void)[] = [];

export const refreshQueryTime = () => {
  queryTimestamp = Date.now();
  subscribers.forEach((callback) => callback(queryTimestamp));
  return queryTimestamp;
};

type QueryTimeContextType = {
  queryTime: number;
  refreshQueryTime: () => number;
};

const QueryTimeContext = createContext<QueryTimeContextType | undefined>(undefined);

/**
 * Provides a shared timestamp reference for data fetching operations.
 *
 * Prevents React Query and other data fetching mechanisms from triggering
 * unnecessary refetches due to components generating new timestamps on each render.
 *
 * Without this utility, each component would:
 * - Create its own `Date.now()` when mounted or re-rendered
 * - Cause React Query to treat it as a new dependency and refetch
 * - Result in different components showing data from inconsistent time windows
 *
 *
 * ```tsx
 * // In a data fetching hook
 * const { queryTime } = useQueryTime();
 *
 * // Use in query parameters
 * const params = {
 *   startTime: queryTime - TIME_WINDOW,
 *   endTime: queryTime
 * };
 *
 * // Only refetches when explicitly refreshed
 * const { data } = useQuery(['data', params], fetchData);
 * ```
 *
 * When you need to refresh all queries simultaneously:
 * ```tsx
 * const { refreshQueryTime } = useQueryTime();
 *
 * // Updates timestamp for ALL components
 * const handleRefresh = () => refreshQueryTime();
 * ```
 */
export const QueryTimeProvider = ({ children }: { children: ReactNode }) => {
  const [queryTime, setQueryTime] = useState(queryTimestamp);

  useEffect(() => {
    const callback = (newTimestamp: number) => setQueryTime(newTimestamp);
    subscribers.push(callback);

    return () => {
      subscribers = subscribers.filter((cb) => cb !== callback);
    };
  }, []);

  return (
    <QueryTimeContext.Provider value={{ queryTime, refreshQueryTime }}>
      {children}
    </QueryTimeContext.Provider>
  );
};

export const useQueryTime = () => {
  const context = useContext(QueryTimeContext);
  if (context === undefined) {
    throw new Error("useQueryTime must be used within a QueryTimeProvider");
  }
  return context;
};
