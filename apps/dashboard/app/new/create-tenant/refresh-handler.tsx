'use client';

import { useEffect } from 'react';
import { useSearchParams } from 'next/navigation';

export function RefreshHandler() {
  const searchParams = useSearchParams();
  
  useEffect(() => {
    if (searchParams && searchParams.get('refresh') === 'true') {
      // Remove the query parameter
      const newUrl = window.location.pathname;
      window.history.replaceState({}, '', newUrl);
      
      // Force a full page refresh
      window.location.reload();
    }
  }, [searchParams]);

  return null;
}