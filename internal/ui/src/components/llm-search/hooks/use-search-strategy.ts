import { useCallback, useRef } from "react";

/**
 * Custom hook that provides different search strategies
 * @param onSearch Function to execute the search
 * @param debounceTime Delay for debounce in ms
 */
export const useSearchStrategy = (onSearch: (query: string) => void, debounceTime = 500) => {
  const debounceTimerRef = useRef<NodeJS.Timeout | null>(null);
  const lastSearchTimeRef = useRef<number>(0);
  const THROTTLE_INTERVAL = 1000;

  /**
   * Clears the debounce timer
   */
  const clearDebounceTimer = useCallback(() => {
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
      debounceTimerRef.current = null;
    }
  }, []);

  /**
   * Executes the search with the given query
   */
  const executeSearch = useCallback(
    (query: string) => {
      if (query.trim()) {
        try {
          lastSearchTimeRef.current = Date.now();
          onSearch(query.trim());
        } catch (error) {
          console.error("Search failed:", error);
        }
      }
    },
    [onSearch],
  );

  /**
   * Debounced search - waits for user to stop typing before executing search
   */
  const debouncedSearch = useCallback(
    (search: string) => {
      clearDebounceTimer();

      debounceTimerRef.current = setTimeout(() => {
        executeSearch(search);
      }, debounceTime);
    },
    [clearDebounceTimer, executeSearch, debounceTime],
  );

  /**
   * Throttled search with initial debounce - debounce first query, throttle subsequent searches
   */

  const throttledSearch = useCallback(
    (search: string) => {
      const now = Date.now();
      const timeElapsed = now - lastSearchTimeRef.current;
      const query = search.trim();

      // If this is the first search, use debounced search
      if (lastSearchTimeRef.current === 0 && query) {
        debouncedSearch(search);
        return;
      }

      // For subsequent searches, use throttling
      if (timeElapsed >= THROTTLE_INTERVAL) {
        // Enough time has passed, execute immediately
        executeSearch(search);
      } else if (query) {
        // Not enough time has passed, schedule for later
        clearDebounceTimer();

        // Schedule execution after remaining throttle time
        const remainingTime = THROTTLE_INTERVAL - timeElapsed;
        debounceTimerRef.current = setTimeout(() => {
          throttledSearch(search);
        }, remainingTime);
      }
    },
    [clearDebounceTimer, debouncedSearch, executeSearch],
  );

  /**
   * Resets search state for new search sequences
   */
  const resetSearchState = useCallback(() => {
    lastSearchTimeRef.current = 0;
  }, []);

  return {
    debouncedSearch,
    throttledSearch,
    executeSearch,
    clearDebounceTimer,
    resetSearchState,
  };
};
