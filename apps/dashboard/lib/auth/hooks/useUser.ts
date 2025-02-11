"use client";

import { useEffect, useRef, useState } from "react";
import { getCurrentUser, listMemberships, refreshSession } from "../actions";
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
  const [user, setUser] = useState<User | null>(null);
  const [memberships, setMemberships] = useState<Membership[]>([]);
  const [metadata, setMetadata] = useState({});

  const [errors, setErrors] = useState<ErrorState>({});
  const [loading, setLoading] = useState<LoadingState>({
    user: true,
    memberships: true,
    switch: false,
  });

  const clearError = (key: keyof ErrorState) => {
    setErrors((prev) => {
      const next = { ...prev };
      delete next[key];
      return next;
    });
  };

  const setError = (key: keyof ErrorState, error: Error) => {
    setErrors((prev) => ({ ...prev, [key]: error }));
  };

  const setLoadingState = (key: keyof LoadingState, isLoading: boolean) => {
    setLoading((prev) => ({ ...prev, [key]: isLoading }));
  };

  const fetchingRef = useRef(false);

  const fetchUser = async () => {
    if (fetchingRef.current) {
      return;
    } 
    fetchingRef.current = true;

    try {
      setLoadingState("user", true);
      clearError("user");
      const userData = await getCurrentUser();
      setUser(userData);
    } catch (err) {
      setError("user", err instanceof Error ? err : new Error("Failed to fetch user"));
    } finally {
      setLoadingState("user", false);
    }
  };

  const fetchMemberships = async (userId?: string) => {
    try {
      setLoadingState("memberships", true);
      clearError("memberships");
      const { data: membershipData, metadata: membershipMetadata } = await listMemberships(userId);
      setMemberships(membershipData);
      setMetadata(membershipMetadata);
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
  };

  const switchOrganization = async (orgId: string) => {
    try {
      setLoadingState("switch", true);
      clearError("switch");
      await refreshSession(orgId);
      await fetchUser(); // user.orgId will change
    } catch (err) {
      setError("switch", err instanceof Error ? err : new Error("Failed to switch organization"));
    } finally {
      setLoadingState("switch", false);
    }
  };

  // Computed states for easier consumption
  const isLoading = Object.values(loading).some(Boolean);
  const hasErrors = Object.keys(errors).length > 0;
  const membership = memberships.find((m) => {
    m.organization.id === user?.orgId;
  });

  useEffect(() => {
      // biome-ignore lint/react-hooks/exhaustiveDeps: only fetching data once on mount, uses a fetchingRef to prevent duplicate calls
    fetchUser();
  }, []);

  useEffect(() => {
    if (user) {
      // biome-ignore lint/react-hooks/exhaustiveDeps: fetchMemberships is stable and only depends on setState functions
      fetchMemberships(user.id);
    }
  }, [user]);

  return {
    // Data
    user,
    memberships,
    membership,
    metadata,

    // Loading states
    loading,
    isLoading,

    // Error states
    errors,
    hasErrors,

    // Actions
    fetchUser,
    fetchMemberships,
    switchOrganization,
  };
}
