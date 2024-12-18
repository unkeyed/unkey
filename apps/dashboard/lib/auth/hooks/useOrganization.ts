import { useState, useEffect } from 'react';
import { useAuth } from '../auth-provider';
import { Membership } from '../interface';

interface UseOrganizationReturn {
    memberships: Membership[];
    metadata: {};
    isLoading: boolean;
    error: Error | null;
    refetch: (userId?: string) => Promise<void>;
    switchOrganization: (orgId: string) => Promise<void>;
    isSwitching: boolean;
  }
  
  export function useOrganization(initialUserId?: string): UseOrganizationReturn {
    const auth = useAuth();
    const [memberships, setMemberships] = useState<Membership[]>([]);
    const [metadata, setMetadata] = useState({});
    const [isLoading, setIsLoading] = useState(true);
    const [isSwitching, setIsSwitching] = useState(false);
    const [error, setError] = useState<Error | null>(null);
  
    const fetchMemberships = async (userId?: string) => {
      try {
        setIsLoading(true);
        setError(null);
        
        const { data: membershipData, metadata: membershipMetadata } = await auth.listMemberships(userId);
        setMemberships(membershipData);
        setMetadata(membershipMetadata);
  
      } catch (err) {
        setError(err instanceof Error ? err : new Error('Failed to fetch memberships'));
        setMemberships([]);
        setMetadata({});
      } finally {
        setIsLoading(false);
      }
    };
  
    const switchOrganization = async (orgId: string) => {
      try {
        setIsSwitching(true);
        setError(null);
        
        await auth.refreshSession(orgId);
      
      } catch (err) {
        setError(err instanceof Error ? err : new Error('Failed to switch organization'));
      } finally {
        setIsSwitching(false);
      }
    };
  
    useEffect(() => {
      if (auth) {
        fetchMemberships(initialUserId);
      }
    }, [auth, initialUserId]);
  
    return {
      memberships,
      metadata,
      isLoading,
      isSwitching,
      error,
      refetch: fetchMemberships,
      switchOrganization
    };
  }