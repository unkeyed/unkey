'use client';

import { createContext, useContext, useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { UNKEY_ACCESS_MAX_AGE } from '@/lib/auth/types';

// Types
type AuthContextType = {
  isAuthenticated: boolean;
  isLoading: boolean;
  refreshSession: () => Promise<boolean>;
};

// Create the context with a default value
const AuthContext = createContext<AuthContextType>({
  isAuthenticated: false,
  isLoading: true,
  refreshSession: async () => false,
});

export function AuthProvider({ 
  children,
  requireAuth = false,
  redirectTo = '/auth/sign-in'
}: { 
  children: React.ReactNode;
  requireAuth?: boolean;
  redirectTo?: string;
}) {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const router = useRouter();

  // Verify authentication status
  const checkAuth = async () => {
    try {
      setIsLoading(true);
      // This endpoint will use server-side cached getAuth
      const response = await fetch('/api/auth/session', {
        credentials: 'include',
      });

      setIsAuthenticated(response.ok);
      
      if (!response.ok && requireAuth) {
        router.push(redirectTo);
      }
    } catch (error) {
      console.error('Auth check failed:', error);
      setIsAuthenticated(false);
      if (requireAuth) {
        router.push(redirectTo);
      }
    } finally {
      setIsLoading(false);
    }
  };

  // Refresh session - this will use mutex-protected refresh
  const refreshSession = async (): Promise<boolean> => {
    try {
      // This endpoint will trigger mutex-protected refresh function
      const response = await fetch('/api/auth/refresh', {
        method: 'POST',
        credentials: 'include',
      });

      setIsAuthenticated(response.ok);
      
      if (!response.ok && requireAuth) {
        router.push(redirectTo);
      }
      
      return response.ok;
    } catch (error) {
      console.error('Session refresh failed:', error);
      setIsAuthenticated(false);
      if (requireAuth) {
        router.push(redirectTo);
      }
      return false;
    }
  };

  // Set up token refresh interval
  useEffect(() => {
    let refreshInterval: NodeJS.Timeout;

    if (isAuthenticated) {
      // Refresh tokens proactively 1 min before expires
      const refreshMaxAge = UNKEY_ACCESS_MAX_AGE - 6000;
      refreshInterval = setInterval(() => {
        refreshSession();
      }, refreshMaxAge);
    }

    return () => {
      if (refreshInterval) clearInterval(refreshInterval);
    };
  }, [isAuthenticated]);

  // Handle multi-tab logout
  useEffect(() => {
    const handleStorageChange = (e: StorageEvent) => {
      if (e.key === 'auth_logout' && isAuthenticated) {
        setIsAuthenticated(false);
        if (requireAuth) {
          router.push(redirectTo);
        }
      }
    };
    
    window.addEventListener('storage', handleStorageChange);
    return () => {
      window.removeEventListener('storage', handleStorageChange);
    };
  }, [isAuthenticated, requireAuth, redirectTo, router]);

  // Initial auth check
  useEffect(() => {
    checkAuth();
  }, []);

  return (
    <AuthContext.Provider
      value={{
        isAuthenticated,
        isLoading,
        refreshSession
      }}
    >
      {!requireAuth || (requireAuth && isAuthenticated) ? children : null}
    </AuthContext.Provider>
  );
}