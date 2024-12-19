import { useAuth } from '../auth-provider';
import { useState, useTransition } from 'react';

export function useSignOut() {
  const { getSignOutUrl } = useAuth();
  const [isLoading, setIsLoading] = useState(false);
  const [isPending, startTransition] = useTransition();

  const handleSignOut = async () => {
    try {
      setIsLoading(true);
      
      startTransition(async () => {
        const signOutUrl = await getSignOutUrl();
        window.location.assign(signOutUrl);
      });
    } catch (error) {
      console.error('Sign out failed:', error);
      setIsLoading(false);
    }
  };

  return {
    signOut: handleSignOut,
    isLoading: isLoading || isPending
  };
}