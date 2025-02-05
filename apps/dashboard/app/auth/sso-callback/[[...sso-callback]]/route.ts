import { NextRequest, NextResponse } from 'next/server';
import { auth } from '@/lib/auth/server';
import { cookies } from 'next/headers';
import { AuthErrorCode, PENDING_SESSION_COOKIE, SIGN_IN_URL } from '@/lib/auth/types';

export async function GET(request: NextRequest) {
    const authResult = await auth.completeOAuthSignIn(request);
    
    if (!authResult.success) {
        if (authResult.code === AuthErrorCode.ORGANIZATION_SELECTION_REQUIRED && authResult.cookies?.[0]) {
            const url = new URL(SIGN_IN_URL, request.url);
            
            // Add orgs to searchParams to make it accessible to the client
            if ('organizations' in authResult) {
                url.searchParams.set('orgs', JSON.stringify(authResult.organizations));
            }
            
            const response = NextResponse.redirect(url);
            
            cookies().set(PENDING_SESSION_COOKIE, authResult.cookies[0].value, {
                secure: true,
                httpOnly: true
            });

            return response;
        }
        
        // Handle other errors
        return NextResponse.redirect(new URL(SIGN_IN_URL, request.url));
    }

    // Get base URL from request because Next.js wants it
    const baseUrl = new URL(request.url).origin;
    const response = NextResponse.redirect(new URL(authResult.redirectTo, baseUrl));

    // Set actual session cookies
    for (const cookie of authResult.cookies) {
        cookies().set(cookie.name, cookie.value, cookie.options);
    }
  
    return response;
}