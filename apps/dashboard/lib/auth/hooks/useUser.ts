import { useState, useEffect } from 'react';
import { auth } from '../index';
import { User } from '../interface';


interface UseUserReturn {
  user: User | null;
  isLoading: boolean;
  error: Error | null;
  refetch: () => Promise<void>;
}

export function useUser(): UseUserReturn {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchUser = async () => {
    try {
      setIsLoading(true);
      setError(null);
      const userData = await auth.getCurrentUser();
      setUser(userData);
    } catch (err) {
      setError(err instanceof Error ? err : new Error('Failed to fetch user'));
      setUser(null);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchUser();
  }, []);

  return {
    user,
    isLoading,
    error,
    refetch: fetchUser
  };
}