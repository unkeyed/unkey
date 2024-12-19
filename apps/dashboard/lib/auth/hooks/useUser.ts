'use client';

import { useState, useEffect, useRef, useTransition } from 'react';
import type { User } from '../types';
import { getCurrentUser } from '../actions';

// useUser hook
export function useUser() {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  const fetchingRef = useRef(false);
  const [isPending, startTransition] = useTransition();

  const fetchUser = async () => {
    if (fetchingRef.current) return;
    fetchingRef.current = true;
    
    try {
      setIsLoading(true);
      setError(null);
      
      startTransition(async () => {
        try {
          const userData = await getCurrentUser();
          setUser(userData);
        } catch (err) {
          setError(err instanceof Error ? err : new Error('Failed to fetch user'));
          setUser(null);
        }
      });
    } finally {
      setIsLoading(false);
      fetchingRef.current = false;
    }
  };

  useEffect(() => {
    fetchUser();
  }, []);

  return { 
    user, 
    isLoading: isLoading || isPending, 
    error, 
    refetch: fetchUser 
  };
}