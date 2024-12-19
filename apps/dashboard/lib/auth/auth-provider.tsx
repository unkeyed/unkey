'use client';

import React, { createContext, useContext, ReactNode } from 'react';
import { useTransition } from 'react';
import type { OrgMembership, OAuthStrategy } from './interface';

interface AuthContextType {
  listMemberships: (userId?: string) => Promise<OrgMembership>;
  refreshSession: (orgId: string) => Promise<void>;
  getCurrentUser: () => Promise<any>; // Replace 'any' with your user type
  getSignOutUrl: () => Promise<string>;
  initiateOAuthSignIn: (params: { 
    provider: OAuthStrategy; 
    redirectUrlComplete: string 
  }) => Promise<{ url: string | null; error?: string }>;
  isPending: boolean;
}

const AuthContext = createContext<AuthContextType | null>(null);

interface AuthProviderProps {
  children: ReactNode;
}

export function AuthProvider({ children }: AuthProviderProps) {
  const [isPending, startTransition] = useTransition();

  // Import actions dynamically to avoid server/client mismatch
  const authService: AuthContextType = {
    listMemberships: async (userId?: string) => {
      const { listMembershipsAction } = await import('./actions');
      return await listMembershipsAction(userId);
    },
    refreshSession: async (orgId: string) => {
      const { refreshSessionAction } = await import('./actions');
      startTransition(async () => {
        await refreshSessionAction(orgId);
      });
    },
    getCurrentUser: async () => {
      const { getCurrentUserAction } = await import('./actions');
      return await getCurrentUserAction();
    },
    getSignOutUrl: async () => {
      const { getSignOutUrlAction } = await import('./actions');
      return await getSignOutUrlAction();
    },
    initiateOAuthSignIn: async (params) => {
      const { initiateOAuthSignIn } = await import('./actions');
      return await initiateOAuthSignIn(params);
    },
    isPending
  };

  return (
    <AuthContext.Provider value={authService}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}