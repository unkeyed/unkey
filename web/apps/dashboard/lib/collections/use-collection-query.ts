"use client";

import {
  QueryObserver,
  type QueryObserverOptions,
  type QueryObserverResult,
} from "@tanstack/query-core";
import { useCallback, useEffect, useState, useSyncExternalStore } from "react";
import { queryClient } from "./client";

// Same as useQuery, but stores the data in the collections' query client, so
// collection pollers can refresh it. Plain useQuery stores it in a different
// client that the pollers never touch
export function useCollectionQuery<T>(options: QueryObserverOptions<T>): QueryObserverResult<T> {
  const [observer] = useState(() => new QueryObserver<T>(queryClient, options));
  useEffect(() => {
    observer.setOptions(options);
  });
  return useSyncExternalStore(
    useCallback((onChange) => observer.subscribe(onChange), [observer]),
    () => observer.getCurrentResult(),
    () => observer.getCurrentResult(),
  );
}
