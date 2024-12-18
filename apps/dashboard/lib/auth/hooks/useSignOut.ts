import { useAuth } from '../auth-provider';
import { useState } from 'react';

export function useSignOut() {
    const auth = useAuth();
    const [isLoading, setIsLoading] = useState(false)
  
    const handleSignOut = async () => {
      try {
        setIsLoading(true)
        const signOutUrl = await auth.getSignOutUrl()
        window.location.assign(signOutUrl);
      } catch (error) {
        console.error('Sign out failed:', error)
        setIsLoading(false)
      }
    }
  
    return {
      signOut: handleSignOut,
      isLoading
    }
  }