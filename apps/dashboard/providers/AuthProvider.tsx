"use client";

import { UNKEY_ACCESS_MAX_AGE } from "@/lib/auth/types";
import { useRouter } from "next/navigation";
import { createContext, useCallback, useEffect, useState } from "react";

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
  redirectTo = "/auth/sign-in",
}: {
  children: React.ReactNode;
  requireAuth?: boolean;
  redirectTo?: string;
}) {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const router = useRouter();

  // Verify authentication status
  const checkAuth = useCallback(async () => {
    try {
      setIsLoading(true);
      // This endpoint will use server-side cached getAuth
      const response = await fetch("/api/auth/session", {
        credentials: "include",
      });

      setIsAuthenticated(response.ok);

      if (!response.ok && requireAuth) {
        router.push(redirectTo);
      }
    } catch (error) {
      console.error("Auth check failed:", error);
      setIsAuthenticated(false);
      if (requireAuth) {
        router.push(redirectTo);
      }
    } finally {
      setIsLoading(false);
    }
  }, [requireAuth, router, redirectTo]);

  // Refresh session - this will use mutex-protected refresh
  const refreshSession = useCallback(async () => {
    try {
      const response = await fetch("/api/auth/refresh", {
        method: "POST",
        credentials: "include",
      });

      setIsAuthenticated(response.ok);

      if (!response.ok && requireAuth) {
        router.push(redirectTo);
      }

      return response.ok;
    } catch (error) {
      console.error("Session refresh failed:", error);
      setIsAuthenticated(false);
      if (requireAuth) {
        router.push(redirectTo);
      }
      return false;
    }
  }, [requireAuth, router, redirectTo]);

  // Set up token refresh interval
  useEffect(() => {
    let refreshInterval: NodeJS.Timeout;

    if (isAuthenticated) {
      refreshInterval = setInterval(refreshSession, UNKEY_ACCESS_MAX_AGE - 6000);
    }

    return () => {
      if (refreshInterval) {
        clearInterval(refreshInterval);
      }
    };
  }, [isAuthenticated, refreshSession]);

  // Initial auth check
  useEffect(() => {
    checkAuth();
  }, [checkAuth]);

  return (
    <AuthContext.Provider
      value={{
        isAuthenticated,
        isLoading,
        refreshSession,
      }}
    >
      {!requireAuth || (requireAuth && isAuthenticated) ? children : null}
    </AuthContext.Provider>
  );
}
