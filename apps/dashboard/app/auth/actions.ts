'use server'

import { auth } from "@/lib/auth/index";
import { OAuthStrategy } from "@/lib/auth/interface";

// this just serves as a flyweight between the pure client-side and the pure server-side auth provider. 
export async function initiateOAuthSignIn({provider, redirectUrlComplete} : {provider: OAuthStrategy, redirectUrlComplete: string}) {
    try {
      return await auth.signInViaOAuth({ 
        provider,
        redirectUrlComplete
      });
  
    } catch (error) {
      console.error('OAuth initialization error:', error);
      return { error: error instanceof Error ? error.message : 'Authentication failed' };
    }
  }