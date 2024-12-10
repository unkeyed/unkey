import { NextRequest, NextResponse } from 'next/server';
import { auth } from '@/lib/auth/index'
import { cookies } from 'next/headers';

export async function GET(request: NextRequest) {
    const authResult = await auth.completeOAuthSignIn(request);

    // Get base URL from request because Next.js wants it
    const baseUrl = new URL(request.url).origin;
    const response = NextResponse.redirect(new URL(authResult.redirectTo, baseUrl));

    // Set actual session cookies
    for (const cookie of authResult.cookies) {
        cookies().set(cookie.name, cookie.value, cookie.options);
    }
  
    return response;
}