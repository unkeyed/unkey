'use server'

import { auth } from "@/lib/auth/index";
import { OAuthStrategy } from "@/lib/auth/interface";

type OAuthSignInResult = {
    url: string | null;
    error?: string;
  }
// this just serves as a flyweight between the pure client-side and the pure server-side auth provider. 
  
export async function initiateOAuthSignIn({
    provider, 
    redirectUrlComplete
  }: {
    provider: OAuthStrategy;
    redirectUrlComplete: string;
  }): Promise<OAuthSignInResult> {
    try {
      const url = auth.signInViaOAuth({ 
        provider,
        redirectUrlComplete
      });
      
      return { url };
  
    } catch (error) {
      console.error('OAuth initialization error:', error);
      return { 
        url: null, 
        error: error instanceof Error ? error.message : 'Authentication failed'
      };
    }
  }

export async function initiateSignOut() {
    try {

        // return a sign out Url to navigate the sign out url
        return await auth.signOut();
    }

    catch(error) {
        console.error("Sign out error:", error);
        return { error: error instanceof Error ? error.message : 'Sign Out failed' };
    }
}