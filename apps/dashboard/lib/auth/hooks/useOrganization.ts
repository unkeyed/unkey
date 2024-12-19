import { useState, useEffect, useTransition } from 'react';
import { Membership } from '../types';
import { listMemberships, refreshSession } from "../actions";

interface UseOrganizationReturn {
    memberships: Membership[];
    metadata: {};
    isLoading: boolean;
    error: Error | null;
    refetch: (userId?: string) => Promise<void>;
    switchOrganization: (orgId: string) => Promise<void>;
    isSwitching: boolean;
  }
  
export function useOrganization(initialUserId?: string) {
  const [memberships, setMemberships] = useState<Membership[]>([]);
  const [metadata, setMetadata] = useState({});
  const [isLoading, setIsLoading] = useState(true);
  const [isSwitching, setIsSwitching] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [isPending, startTransition] = useTransition();

  const fetchMemberships = async (userId?: string) => {
    try {
      setIsLoading(true);
      setError(null);
      
      startTransition(async () => {
        try {
          const { data: membershipData, metadata: membershipMetadata } = await listMemberships(userId);
          setMemberships(membershipData);
          setMetadata(membershipMetadata);
        } catch (err) {
          setError(err instanceof Error ? err : new Error('Failed to fetch memberships'));
          setMemberships([]);
          setMetadata({});
        }
      });
    } finally {
      setIsLoading(false);
    }
  };

  const switchOrganization = async (orgId: string) => {
    try {
      setIsSwitching(true);
      setError(null);
      
      startTransition(async () => {
        try {
          await refreshSession(orgId);
        } catch (err) {
          setError(err instanceof Error ? err : new Error('Failed to switch organization'));
        }
      });
    } finally {
      setIsSwitching(false);
    }
  };

  useEffect(() => {
    fetchMemberships(initialUserId);
  }, [initialUserId]);

  return {
    memberships,
    metadata,
    isLoading: isLoading || isPending,
    isSwitching: isSwitching || isPending,
    error,
    refetch: fetchMemberships,
    switchOrganization
  };
}