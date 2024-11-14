import { type MagicAuth, User, WorkOS } from "@workos-inc/node";
import { AuthSession, BaseAuthProvider, type SignInViaOAuthOptions } from "./interface";
import { NextResponse } from "next/server";
import { env } from "@/lib/env";

const SSO_CALLBACK_URI = "/auth/sso-callback";
export class WorkOSAuthProvider<T> extends BaseAuthProvider {
  private static instance: WorkOSAuthProvider<any> | null = null;
  private static provider: WorkOS;
  private static clientId: string;

  constructor(config: {apiKey: string, clientId: string}) {
    super();
    if (WorkOSAuthProvider.instance) {
      return WorkOSAuthProvider.instance;
    }

    WorkOSAuthProvider.clientId = config.clientId;
    WorkOSAuthProvider.provider = new WorkOS(config.apiKey);
    WorkOSAuthProvider.instance = this;
  }

  async getOrgId(): Promise<T> {
    // Implementation to get the organization ID
    // If none, trigger a redirect to the sign-in page
    throw new Error("Method not implemented.");
  }

  async getSession(token: string): Promise<AuthSession | null> {
    if (!token) return null;

    const WORKOS_COOKIE_PASSWORD = env().WORKOS_COOKIE_PASSWORD;
    if (!WORKOS_COOKIE_PASSWORD) {
      throw new Error("WORKOS_COOKIE_PASSWORD is required");
    }

    try {
      // Load the sealed session
      const session = WorkOSAuthProvider.provider.userManagement.loadSealedSession({
        sessionData: token,
        cookiePassword: WORKOS_COOKIE_PASSWORD
      });

      // Authenticate the session
      const authResult = await session.authenticate();

      if (authResult.authenticated) {
        const { user, organizationId } = authResult;

        return {
          userId: user.id,
          orgId: organizationId || ''
        };
        
      } else {
        console.debug('Authentication failed:', authResult.reason);
        return null;
      }

    } catch (error) {
      console.error('Session validation error:', {
        error: error instanceof Error ? error.message : 'Unknown error',
        token: token.substring(0, 10) + '...' // Log only part of the token for debugging
      });

      return null;
    }
  }

  async getUser(): Promise<any | null> {
    // Implementation to get the user data
    try {
      
    } catch (error) {
      console.error('Error getting user:', error);
      return;
    }
  }

  async listOrganizations(): Promise<T> {
    // Implementation to list organizations
    throw new Error("Method not implemented.");
  }

  async signUpViaEmail(email: string): Promise<MagicAuth> {
    if (!email) {
      throw new Error("No email address provided.");
    }

    const magicAuth = await WorkOSAuthProvider.provider.userManagement.createMagicAuth({ email });

    return magicAuth;
  }

  async signIn(orgId?: string): Promise<T> {
    // Implementation to sign in the user
    throw new Error("Method not implemented.");
  }

  signInViaOAuth({ 
    redirectUri = SSO_CALLBACK_URI,  // Default value
    provider 
  }: SignInViaOAuthOptions): NextResponse {
    try {
      // Validate provider
      if (!provider) {
        throw new Error('Provider is required');
      }

      const authorizationUrl = WorkOSAuthProvider.provider.userManagement.getAuthorizationUrl({
        clientId: WorkOSAuthProvider.clientId,
        redirectUri,
        provider: provider === "github" ? "GitHubOAuth" : "GoogleOAuth"
      });

      // Redirect to the authorization URL
      return NextResponse.redirect(authorizationUrl);

    } catch (error) {
      console.error('OAuth initialization error:', error);
      throw error;
    }
  }


  async signOut(): Promise<T> {
    // Implementation to sign out the user
    throw new Error("Method not implemented.");
  }

  async updateOrg(org: Partial<T>): Promise<T> {
    // Implementation to update the organization
    throw new Error("Method not implemented.");
  }
}
