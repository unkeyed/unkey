"use client";

import { useEffect, useRef, useState } from "react";
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

  const fetchMemberships = async () => {
    try {
      setLoadingState("memberships", true);
      clearError("memberships");
      const { data: membershipData, metadata: membershipMetadata } = await listMemberships();
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
      
      const result = await switchOrg(orgId);
      
      if (result.success) {
        // Refresh the page to update the app with the new organization context
        window.location.reload();
      } else {
        throw new Error(result.error || "Failed to switch organization");
      }
    } catch (err) {
      setError(
        "switch", 
        err instanceof Error ? err : new Error("Failed to switch organization")
      );
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

  // biome-ignore lint/correctness/useExhaustiveDependencies(fetchUser): fetch user data once on mount
  useEffect(() => {
    fetchUser();
  }, []);

  // biome-ignore lint/correctness/useExhaustiveDependencies(fetchMemberships): fetchMemberships is stable and only depends on setState functions
  useEffect(() => {
    if (user) {
      fetchMemberships();
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
