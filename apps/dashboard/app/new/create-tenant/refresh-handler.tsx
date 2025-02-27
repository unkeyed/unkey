'use client';

import { useEffect } from 'react';
import { useSearchParams } from 'next/navigation';

export function RefreshHandler() {
  const searchParams = useSearchParams();
  
  useEffect(() => {
    const newOrg = searchParams && searchParams.get('refresh') === 'true';
    
    if (newOrg) {
      // Remove the refresh parameter from the URL
      const newUrl = new URL(window.location.href);
      newUrl.searchParams.delete('refresh');
      window.history.replaceState({}, '', newUrl.toString());
      
      // Force a page refresh to ensure we're using the new org context
      window.location.reload();
    }
  }, [searchParams]);

  return null;
}