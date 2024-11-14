'use server'

import { auth } from "@/lib/auth/index";
import { OAuthStrategy } from "@/lib/auth/interface";

export async function initiateOAuthSignIn(provider: OAuthStrategy) {
    try {
      // WorkOS - will redirect internally
      // No Auth - 
      // X Provider - should handle internally, whether its a redirect to an authorization url or returning a Response
      const response = auth.signInViaOAuth({ 
        provider,
        redirectUri: '/auth/sso-callback'
      });
  
      // If the provider's implementation returns a Response, use it
      return response;
  
    } catch (error) {
      console.error('OAuth initialization error:', error);
      return { error: error instanceof Error ? error.message : 'Authentication failed' };
    }
  }