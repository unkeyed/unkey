"use client";
import { useEffect, useState, useMemo } from "react";
import {
  getCurrentUser,
  getInvitationList,
  getOrg,
  getOrganizationMemberList,
  removeMembership,
  revokeOrgInvitation,
  updateMembership,
} from "../actions";
import type { Invitation, Membership, Organization, UpdateMembershipParams } from "../types";

type ErrorState = {
  organization?: Error;
  memberships?: Error;
  invitations?: Error;
  removeMember?: Error;
  revokeInvitation?: Error;
};

type LoadingState = {
  organization: boolean;
  memberships: boolean;
  invitations: boolean;
};

export function useOrganization() {
  const [memberships, setMemberships] = useState<Membership[]>([]);
  const [membershipMetadata, setMembershipMetadata] = useState<Record<string, unknown>>(() => ({}));

  const [invitations, setInvitations] = useState<Invitation[]>([]);
  const [invitationMetadata, setInvitationMetadata] = useState<Record<string, unknown>>(() => ({}));

  // Track if initial fetch is complete to prevent repeated calls
  const [initialFetchComplete, setInitialFetchComplete] = useState(false);
  
  const [organization, setOrganization] = useState<Organization | null>(null);

  const [errors, setErrors] = useState<ErrorState>(() => ({}));
  const [loading, setLoading] = useState<LoadingState>({
    organization: true,
    memberships: true,
    invitations: true,
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

  const fetchMemberships = async (orgId: string) => {
    try {
      setLoadingState("memberships", true);
      clearError("memberships");

      const { data: membershipData, metadata: membershipMeta } = await getOrganizationMemberList(orgId);
      setMemberships(membershipData);
      setMembershipMetadata(membershipMeta);
    } catch (err) {
      setError(
        "memberships",
        err instanceof Error ? err : new Error("Failed to fetch memberships"),
      );
      setMemberships([]);
      setMembershipMetadata({});
    } finally {
      setLoadingState("memberships", false);
    }
  };

  const fetchInvitations = async (orgId: string) => {
    try {
      setLoadingState("invitations", true);
      clearError("invitations");

      const { data, metadata } = await getInvitationList(orgId);
      setInvitations(data);
      setInvitationMetadata(metadata);
    } catch (err) {
      setError(
        "invitations",
        err instanceof Error ? err : new Error("Failed to fetch invitations"),
      );
      setInvitations([]);
      setInvitationMetadata({});
    } finally {
      setLoadingState("invitations", false);
    }
  };

  const fetchOrganization = async () => {
    // Prevent redundant fetches
    if (initialFetchComplete && !isLoading) {
      return;
    }
    
    try {
      setLoadingState("organization", true);
      clearError("organization");

      const user = await getCurrentUser();
      if (!user?.orgId) {
        throw new Error("No organization ID found");
      }

      const organizationData = await getOrg(user.orgId);
      setOrganization(organizationData);

      await Promise.all([
        fetchMemberships(user.orgId),
        fetchInvitations(user.orgId)
      ]);
      
      // Mark initial fetch as complete
      setInitialFetchComplete(true);
    } catch (err) {
      setError(
        "organization",
        err instanceof Error ? err : new Error("Failed to fetch organization"),
      );
      setOrganization(null);
    } finally {
      setLoadingState("organization", false);
    }
  };

  const updateMember = async ({ membershipId, role }: UpdateMembershipParams) => {
    if (!membershipId) {
      throw new Error("Membership Id is required");
    }

    if (!role) {
      throw new Error("Role is required");
    }

    try {
      if (!loading.organization && organization) {
        await updateMembership({ membershipId, orgId: organization.id, role });
        // refetch memberships
        fetchMemberships(organization.id);
      }
    } catch (err) {
      setError(
        "removeMember",
        err instanceof Error ? err : new Error("Failed to remove membership"),
      );
    }
  };
  
  const removeMember = async (membershipId: string) => {
    if (!membershipId) {
      throw new Error("Membership Id is required");
    }

    try {
      if (!loading.organization && organization) {
        await removeMembership({ membershipId, orgId: organization.id });
        // refetch memberships
        fetchMemberships(organization.id);
      }
    } catch (err) {
      setError(
        "removeMember",
        err instanceof Error ? err : new Error("Failed to remove membership"),
      );
    }
  };

  const revokeInvitation = async (invitationId: string) => {
    if (!invitationId) {
      throw new Error("Invitation Id is required");
    }

    try {
      if (!loading.organization && organization) {
        await revokeOrgInvitation({ invitationId, orgId: organization.id });
        // refetch invitations
        fetchInvitations(organization.id);
      }
    } catch (err) {
      setError(
        "revokeInvitation",
        err instanceof Error ? err : new Error("Failed to revoke invitation"),
      );
    }
  };

  // Use a ref to track initial mounting to prevent duplicate calls
  const isMounted = useMemo(() => ({ current: false }), []);
  
  useEffect(() => {
    // Only run once when component mounts
    if (!isMounted.current) {
      isMounted.current = true;
      fetchOrganization();
    }
  }, []);

  // Memoize derived values
  const memoizedMemberships = useMemo(() => memberships, [memberships]);
  const memoizedMembershipMetadata = useMemo(() => membershipMetadata, [membershipMetadata]);
  const memoizedInvitations = useMemo(() => invitations, [invitations]);
  const memoizedInvitationMetadata = useMemo(() => invitationMetadata, [invitationMetadata]);
  
  const isLoading = useMemo(() => Object.values(loading).some(Boolean), [loading]);
  const hasErrors = useMemo(() => Object.keys(errors).length > 0, [errors]);

  return {
    // Data
    organization,
    memberships: memoizedMemberships,
    membershipMetadata: memoizedMembershipMetadata,
    invitations: memoizedInvitations,
    invitationMetadata: memoizedInvitationMetadata,

    // Loading states
    loading,
    isLoading,

    // Error states
    errors,
    hasErrors,

    // Actions
    refetchOrganization: () => {
      setInitialFetchComplete(false); // Reset the flag to allow a new fetch
      return fetchOrganization();
    },
    refetchMemberships: () => organization?.id && fetchMemberships(organization.id),
    refetchInvitations: () => organization?.id && fetchInvitations(organization.id),
    removeMember,
    revokeInvitation,
    updateMember,
  };
}