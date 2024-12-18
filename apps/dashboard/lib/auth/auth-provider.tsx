'use client';

import React, { createContext, useContext, ReactNode, useMemo } from 'react';
import { 
  listMembershipsAction, 
  refreshSessionAction, 
  getCurrentUserAction, 
  getSignOutUrlAction 
} from './actions';

interface AuthContextType {
  listMemberships: typeof listMembershipsAction;
  refreshSession: typeof refreshSessionAction;
  getCurrentUser: typeof getCurrentUserAction;
  getSignOutUrl: typeof getSignOutUrlAction;
}

const AuthContext = createContext<AuthContextType | null>(null);

interface AuthProviderProps {
  children: ReactNode;
}

export function AuthProvider({ children }: AuthProviderProps) {
  const authService = useMemo<AuthContextType>(() => ({
    listMemberships: listMembershipsAction,
    refreshSession: refreshSessionAction,
    getCurrentUser: getCurrentUserAction,
    getSignOutUrl: getSignOutUrlAction
  }), []); // Empty deps array since actions are stable

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