"use client";

import { useEffect, useRef, useState, useMemo, useCallback } from "react";
import { getCurrentUser, listMemberships, switchOrg } from "../actions";
import type { Membership, User } from "../types";

type ErrorState = {
  user?: Error;
  memberships?: Error;
  switch?: Error;
};

type LoadingState = {
  user: boolean;
  memberships: boolean;
  switch: boolean;
};

// useUser hook
export function useUser() {
  // State
  const [user, setUser] = useState<User | null>(null);
  const [memberships, setMemberships] = useState<Membership[]>([]);
  const [metadata, setMetadata] = useState<Record<string, unknown>>({});
  const [initialFetchComplete, setInitialFetchComplete] = useState(false);

  const [errors, setErrors] = useState<ErrorState>({});
  const [loading, setLoading] = useState<LoadingState>({
    user: true,
    memberships: true,
    switch: false,
  });

  // Prevent duplicate fetches
  const fetchingRef = useRef(false);
  const userFetchedRef = useRef(false);
  const membershipsFetchedRef = useRef(false);

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

  // Fetch functions
  const fetchUser = useCallback(async () => {
    // Prevent duplicate fetches
    if (fetchingRef.current || userFetchedRef.current) {
      return;
    }

    fetchingRef.current = true;

    try {
      setLoadingState("user", true);
      clearError("user");

      const userData = await getCurrentUser();
      setUser(userData);

      // Mark user as fetched
      userFetchedRef.current = true;
    } catch (err) {
      setError("user", err instanceof Error ? err : new Error("Failed to fetch user"));
    } finally {
      setLoadingState("user", false);
      fetchingRef.current = false;
    }
  }, [clearError, setError, setLoadingState]);

  const fetchMemberships = useCallback(async () => {
    // Skip if already fetched or user is not available
    if (membershipsFetchedRef.current || !user) {
      return;
    }

    try {
      setLoadingState("memberships", true);
      clearError("memberships");

      const { data: membershipData, metadata: membershipMetadata } = await listMemberships();
      setMemberships(membershipData);
      setMetadata(membershipMetadata || {});

      // Mark memberships as fetched
      membershipsFetchedRef.current = true;

      // Mark initial fetch as complete after both user and memberships are loaded
      if (userFetchedRef.current) {
        setInitialFetchComplete(true);
      }
    } catch (err) {
      setError(
        "memberships",
        err instanceof Error ? err : new Error("Failed to fetch memberships"),
      );
      setMemberships([]);
      setMetadata({});
    } finally {
      setLoadingState("memberships", false);
    }
  }, [user, clearError, setError, setLoadingState]);

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

  // Initial data fetching
  const isMounted = useMemo(() => ({ current: false }), []);

  useEffect(() => {
    // Only run once when component mounts
    if (!isMounted.current) {
      isMounted.current = true;
      fetchUser();
    }
  }, [fetchUser]);

  useEffect(() => {
    if (user && !membershipsFetchedRef.current) {
      fetchMemberships();
    }
  }, [user, fetchMemberships]);

  // Memoized values
  const memoizedUser = useMemo(() => user, [user]);
  const memoizedMemberships = useMemo(() => memberships, [memberships]);
  const memoizedMetadata = useMemo(() => metadata, [metadata]);

  const membership = useMemo(() => {
    return memoizedMemberships.find((m) => m.organization.id === memoizedUser?.orgId) || null;
  }, [memoizedMemberships, memoizedUser]);

  const isLoading = useMemo(() => Object.values(loading).some(Boolean), [loading]);

  const hasErrors = useMemo(() => Object.keys(errors).length > 0, [errors]);

  // Reset fetch flags to allow refetching
  const resetFetchState = useCallback(() => {
    userFetchedRef.current = false;
    membershipsFetchedRef.current = false;
    setInitialFetchComplete(false);
  }, []);

  // Refetch methods
  const refetchUser = useCallback(() => {
    userFetchedRef.current = false;
    return fetchUser();
  }, [fetchUser]);

  const refetchMemberships = useCallback(() => {
    membershipsFetchedRef.current = false;
    return fetchMemberships();
  }, [fetchMemberships]);

  return {
    // Data
    user: memoizedUser,
    memberships: memoizedMemberships,
    membership,
    metadata: memoizedMetadata,

    // Loading states
    loading,
    isLoading,

    // Error states
    errors,
    hasErrors,

    // Actions
    fetchUser: refetchUser,
    fetchMemberships: refetchMemberships,
    switchOrganization,
  };
}
