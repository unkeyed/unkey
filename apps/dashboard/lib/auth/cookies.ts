'use server'
import { cookies } from 'next/headers';

export async function handleSessionRefresh(
    cookieName: string,
    sessionValue: string | null | undefined,
    reason?: string
  ): Promise<void> {
      const cookieStore = cookies();
      
      if (sessionValue) {
          cookieStore.set(cookieName, sessionValue, {
              httpOnly: true,
              secure: true
          });
          return;
      }
      
      if (reason) {
          console.debug('Session refresh failed:', reason);
          cookieStore.delete(cookieName);
      }
  }