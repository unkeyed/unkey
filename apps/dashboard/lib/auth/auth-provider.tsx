'use client';

import React, { createContext, useContext, ReactNode } from 'react';
import { listMembershipsAction, refreshSessionAction, getCurrentUserAction } from './actions';

interface AuthContextType {
  listMemberships: typeof listMembershipsAction;
  refreshSession: typeof refreshSessionAction;
  getCurrentUser: typeof getCurrentUserAction;
}

const AuthContext = createContext<AuthContextType | null>(null);

interface AuthProviderProps {
  children: ReactNode;
}

export function AuthProvider({ children }: AuthProviderProps) {
  const authService: AuthContextType = {
    listMemberships: listMembershipsAction,
    refreshSession: refreshSessionAction,
    getCurrentUser: getCurrentUserAction
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