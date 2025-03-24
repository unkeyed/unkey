"use client";

import { useState, useMemo, useCallback } from "react";
import { switchOrg } from "../actions";

type ErrorState = {
  switch?: Error;
};

type LoadingState = {
  switch: boolean;
};

// useUser hook
export function useUser() {
  // State
  const [errors, setErrors] = useState<ErrorState>({});
  const [loading, setLoading] = useState<LoadingState>({
    switch: false,
  });

  // Error handling utilities
  const clearError = useCallback((key: keyof ErrorState) => {
    setErrors((prev) => {
      const next = { ...prev };
      delete next[key];
      return next;
    });
  }, []);

  const setError = useCallback((key: keyof ErrorState, error: Error) => {
    setErrors((prev) => ({ ...prev, [key]: error }));
  }, []);

  const setLoadingState = useCallback((key: keyof LoadingState, isLoading: boolean) => {
    setLoading((prev) => ({ ...prev, [key]: isLoading }));
  }, []);


  const switchOrganization = useCallback(
    async (orgId: string) => {
      try {
        setLoadingState("switch", true);
        clearError("switch");

        const result = await switchOrg(orgId);

        if (result.success) {
          // Don't refresh the location if it's on the creation page because you will lose context
          // due to needed to switch first and then push the parameters after
          if (window.location.pathname !== "/new") {
            window.location.reload();
          }
        } else {
          throw new Error(result.error || "Failed to switch organization");
        }
      } catch (err) {
        setError("switch", err instanceof Error ? err : new Error("Failed to switch organization"));
      } finally {
        setLoadingState("switch", false);
      }
    },
    [clearError, setError, setLoadingState],
  );

  const isLoading = useMemo(() => Object.values(loading).some(Boolean), [loading]);

  const hasErrors = useMemo(() => Object.keys(errors).length > 0, [errors]);

  return {
    // Loading states
    loading,
    isLoading,

    // Error states
    errors,
    hasErrors,

    // Actions
    switchOrganization,
  };
}
