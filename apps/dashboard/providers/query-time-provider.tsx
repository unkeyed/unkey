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
