'use server'
import { cookies } from 'next/headers';
import { NextRequest, NextResponse } from 'next/server';

export interface CookieOptions {
  secure?: boolean;
  httpOnly?: boolean;
  sameSite?: 'lax' | 'strict' | 'none';
  path?: string;
}

export interface Cookie {
  name: string;
  value: string;
  options?: CookieOptions;
}

export class CookieService {
  /**
   * Get a cookie value by name
   */
  static getCookie(name: string, request?: NextRequest): string | null {
    const cookieStore = request?.cookies || cookies();
    return cookieStore.get(name)?.value ?? null;
  }

  /**
   * Set a cookie with the given name, value, and options
   */
  static setCookie(cookie: Cookie): void {
    const cookieStore = cookies();
    cookieStore.set(cookie.name, cookie.value, cookie.options);
  }

  /**
   * Set multiple cookies at once
   */
  static setCookies(cookieList: Cookie[]): void {
    const cookieStore = cookies();
    for (const cookie of cookieList) {
      cookieStore.set(cookie.name, cookie.value, cookie.options);
    }
  }

  /**
   * Delete a cookie by name
   */
  static deleteCookie(name: string): void {
    const cookieStore = cookies();
    cookieStore.delete(name);
  }

  /**
   * Update or clear a secure HTTP-only cookie with optional deletion logging
   * @param cookieName - Name of the cookie to update/clear
   * @param value - Value to set (if null/undefined, cookie will be deleted)
   * @param reason - Optional reason for deletion (will be logged)
   */
  static async updateCookie(
    cookieName: string,
    value: string | null | undefined,
    reason?: string
  ): Promise<void> {
    if (value) {
      this.setCookie({
        name: cookieName,
        value: value,
        options: {
          httpOnly: true,
          secure: true
        }
      });
      return;
    }
    
    if (reason) {
      console.debug('Cookie update failed:', reason);
      this.deleteCookie(cookieName);
    }
  }

  /**
   * Set cookies on a NextResponse object
   * Useful when you need to set cookies during a redirect
   */
  static setCookiesOnResponse(response: NextResponse, cookieList: Cookie[]): NextResponse {
    for (const cookie of cookieList) {
      response.cookies.set(cookie.name, cookie.value, cookie.options);
    }
    return response;
  }
}