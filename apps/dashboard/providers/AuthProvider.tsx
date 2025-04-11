"use client";

import { UNKEY_USER_IDENTITY_COOKIE, UNKEY_USER_IDENTITY_MAX_AGE } from "@/lib/auth/types";
import { useRouter } from "next/navigation";
import { createContext, useCallback, useEffect, useState } from "react";

// Types
type AuthContextType = {
  isAuthenticated: boolean;
  isLoading: boolean;
  accessToken: string | null;
  expiresAt: Date | null;
  refreshSession: () => Promise<boolean>;
};

// Create the context with a default value
const AuthContext = createContext<AuthContextType>({
  isAuthenticated: false,
  isLoading: true,
  accessToken: null,
  expiresAt: null,
  refreshSession: async () => false,
});

// Helper function to get a cookie client-side
const getCookie = (name: string): string | null => {
  const cookies = document.cookie.split(";");
  for (let i = 0; i < cookies.length; i++) {
    const cookie = cookies[i].trim();
    if (cookie.startsWith(`${name}=`)) {
      return cookie.substring(name.length + 1);
    }
  }
  return null;
};

// Helper function to set a cookie client-side
const setCookie = ({
  name,
  value,
  maxAge,
}: { name: string; value: string; maxAge: number }): void => {
  document.cookie = `${name}=${value}; path=/; max-age=${maxAge}; samesite=strict; secure=${window.location.protocol === "https:"}`;
};

export function AuthProvider({
  children,
  requireAuth = false,
  redirectTo = "/auth/sign-in",
  serverGeneratedIdentity,
}: {
  children: React.ReactNode;
  requireAuth?: boolean;
  redirectTo?: string;
  serverGeneratedIdentity: string;
}) {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [accessToken, setAccessToken] = useState<string | null>(null);
  const [expiresAt, setExpiresAt] = useState<Date | null>(null);
  const router = useRouter();

  // Helper to ensure a user identity exists
  const ensureUserIdentity = useCallback((): string => {
    // First check for existing cookie
    let userIdentity = getCookie(UNKEY_USER_IDENTITY_COOKIE);

    // If we don't have an identity cookie yet, use the server-generated one or create a new one
    if (!userIdentity) {
      // Use server-generated identity if provided
      userIdentity = serverGeneratedIdentity;
    }

    // Store for 90 days
    setCookie({
      name: UNKEY_USER_IDENTITY_COOKIE,
      value: userIdentity,
      maxAge: UNKEY_USER_IDENTITY_MAX_AGE,
    });

    return userIdentity;
  }, [serverGeneratedIdentity]);

  // Ensure we have a user identity as early as possible
  useEffect(() => {
    ensureUserIdentity();
  }, [ensureUserIdentity]);

  // Verify authentication status
  const checkAuth = useCallback(async () => {
    try {
      setIsLoading(true);
      // This endpoint will use server-side cached getAuth
      const response = await fetch("/api/auth/session", {
        credentials: "include",
      });

      if (response.ok) {
        const sessionData = await response.json();
        setIsAuthenticated(true);

        // Store access token in memory for easy access by client components
        if (sessionData.accessToken) {
          setAccessToken(sessionData.accessToken);
          setExpiresAt(sessionData.expiresAt || null);
        }
      } else {
        setIsAuthenticated(false);
        setAccessToken(null);
        setExpiresAt(null);

        if (requireAuth) {
          router.push(redirectTo);
        }
      }
    } catch (error) {
      console.error("Auth check failed:", error);
      setIsAuthenticated(false);
      setAccessToken(null);
      setExpiresAt(null);

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
      const userIdentity = ensureUserIdentity();

      const response = await fetch("/api/auth/refresh", {
        method: "POST",
        credentials: "include",
        headers: {
          // Include the user identity in the request headers
          "x-user-identity": userIdentity,
        },
      });

      if (response.ok) {
        const refreshData = await response.json();
        setIsAuthenticated(true);

        // Update access token in memory
        if (refreshData.accessToken) {
          setAccessToken(refreshData.accessToken);
          setExpiresAt(refreshData.expiresAt || null);
        }

        return true;
      }
      setIsAuthenticated(false);
      setAccessToken(null);
      setExpiresAt(null);

      if (requireAuth) {
        router.push(redirectTo);
      }

      return false;
    } catch (error) {
      console.error("Session refresh failed:", error);
      setIsAuthenticated(false);
      setAccessToken(null);
      setExpiresAt(null);

      if (requireAuth) {
        router.push(redirectTo);
      }

      return false;
    }
  }, [requireAuth, router, redirectTo, ensureUserIdentity]);

  // Determine if we need to refresh the token
  const shouldRefreshToken = useCallback(() => {
    if (!expiresAt) {
      return true;
    }

    // Refresh if we're within 1 minute of expiration
    const now = new Date();
    const timeRemaining = expiresAt.getTime() - now.getTime();
    return timeRemaining < 60 * 1000; // Less than 1 minute remaining
  }, [expiresAt]);

  // Set up token refresh interval
  useEffect(() => {
    let refreshInterval: NodeJS.Timeout;

    if (isAuthenticated) {
      // Check tokens periodically and refresh if needed
      refreshInterval = setInterval(() => {
        if (shouldRefreshToken()) {
          refreshSession();
        }
      }, 60 * 1000); // Check every minute
    }

    return () => {
      if (refreshInterval) {
        clearInterval(refreshInterval);
      }
    };
  }, [isAuthenticated, refreshSession, shouldRefreshToken]);

  // Add a refresh listener for when the tab becomes visible again
  // helps with token expiration during long periods of inactivity
  useEffect(() => {
    const handleVisibilityChange = () => {
      if (document.visibilityState === "visible" && isAuthenticated && shouldRefreshToken()) {
        refreshSession();
      }
    };

    document.addEventListener("visibilitychange", handleVisibilityChange);

    return () => {
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    };
  }, [isAuthenticated, refreshSession, shouldRefreshToken]);

  // Initial auth check
  useEffect(() => {
    checkAuth();
  }, [checkAuth]);

  return (
    <AuthContext.Provider
      value={{
        isAuthenticated,
        isLoading,
        accessToken,
        expiresAt,
        refreshSession,
      }}
    >
      {!requireAuth || (requireAuth && isAuthenticated) ? children : null}
    </AuthContext.Provider>
  );
}

export { AuthContext };
