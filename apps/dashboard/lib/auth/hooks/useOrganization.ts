"use client";
import { useEffect, useState } from "react";
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

  const fetchOrganization = async () => {
    try {
      setLoadingState("organization", true);
      clearError("organization");

      const user = await getCurrentUser();
      if (!user?.orgId) {
        throw new Error("No organization ID found");
      }

      const organizationData = await getOrg(user.orgId);
      setOrganization(organizationData);

      // Fetch both lists after we have the organization
      await Promise.all([fetchMemberships(), fetchInvitations()]);
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

  const fetchMemberships = async () => {
    if (!organization?.id) {
      return;
    }

    try {
      setLoadingState("memberships", true);
      clearError("memberships");

      const { data: membershipData, metadata: membershipMetadata } =
        await getOrganizationMemberList(organization.id);
      setMemberships(membershipData);
      setMembershipMetadata(membershipMetadata);
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

  const fetchInvitations = async () => {
    if (!organization?.id) {
      return;
    }

    try {
      setLoadingState("invitations", true);
      clearError("invitations");

      const { data, metadata } = await getInvitationList(organization.id);
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
        fetchMemberships();
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
        fetchMemberships();
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
        fetchInvitations();
      }
    } catch (err) {
      setError(
        "revokeInvitation",
        err instanceof Error ? err : new Error("Failed to revoke invitation"),
      );
    }
  };

  // biome-ignore lint/correctness/useExhaustiveDependencies(fetchOrganization): fetch organization data once on mount
  useEffect(() => {
    fetchOrganization();
  }, []);

  const isLoading = Object.values(loading).some(Boolean);
  const hasErrors = Object.keys(errors).length > 0;

  return {
    // Data
    organization,
    memberships,
    membershipMetadata,
    invitations,
    invitationMetadata,

    // Loading states
    loading,
    isLoading,

    // Error states
    errors,
    hasErrors,

    // Actions
    refetchOrganization: fetchOrganization,
    refetchMemberships: fetchMemberships,
    refetchInvitations: fetchInvitations,
    removeMember,
    revokeInvitation,
    updateMember,
  };
}
