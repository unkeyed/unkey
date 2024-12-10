import { type MagicAuth, WorkOS } from "@workos-inc/node";
import { AuthSession, BaseAuthProvider, OAuthResult, Organization, UNKEY_SESSION_COOKIE, type SignInViaOAuthOptions } from "./interface";
import { env } from "@/lib/env";
import { handleSessionRefresh } from "./cookies";

const SIGN_IN_REDIRECT = "/apis";
const SIGN_IN_URL = "/auth/sign-in";

export class WorkOSAuthProvider extends BaseAuthProvider {
  private static instance: WorkOSAuthProvider | null = null;
  private static provider: WorkOS;
  private static clientId: string;

  constructor(config: { apiKey: string; clientId: string }) {
    super();
    if (WorkOSAuthProvider.instance) {
      return WorkOSAuthProvider.instance;
    }

    WorkOSAuthProvider.clientId = config.clientId;
    WorkOSAuthProvider.provider = new WorkOS(config.apiKey, { clientId: config.clientId });
    WorkOSAuthProvider.instance = this;
  }

  async validateSession(): Promise<AuthSession | null> {
    const token = await this.getSession();
    if (!token) return null;

    const WORKOS_COOKIE_PASSWORD = env().WORKOS_COOKIE_PASSWORD;
    if (!WORKOS_COOKIE_PASSWORD) {
      throw new Error("WORKOS_COOKIE_PASSWORD is required");
    }

    try {
      const session = await WorkOSAuthProvider.provider.userManagement.loadSealedSession({
        sessionData: token,
        cookiePassword: WORKOS_COOKIE_PASSWORD
      });

      const authResult = await session.authenticate();

      if (authResult.authenticated) {
        return {
          userId: authResult.user.id,
          orgId: authResult.organizationId || null,
        };
      }

      console.debug('Authentication failed:', authResult.reason);
      return null;

    } catch (error) {
      console.error('Session validation error:', {
        error: error instanceof Error ? error.message : 'Unknown error',
        token: token.substring(0, 10) + '...'
      });
      return null;
    }
  }

  protected async refreshSession(orgId?: string): Promise<any | null> {
    const token = await this.getSession();
    if (!token) return null;

    const WORKOS_COOKIE_PASSWORD = env().WORKOS_COOKIE_PASSWORD;
    if (!WORKOS_COOKIE_PASSWORD) {
      throw new Error("WORKOS_COOKIE_PASSWORD is required");
    }

    try {
      const session = await WorkOSAuthProvider.provider.userManagement.loadSealedSession({
        sessionData: token,
        cookiePassword: WORKOS_COOKIE_PASSWORD
      });

      const refreshResult = await session.refresh({
        cookiePassword: WORKOS_COOKIE_PASSWORD,
        ...(orgId && { organizationId: orgId })
      });

      if (refreshResult.authenticated) {
        await handleSessionRefresh(UNKEY_SESSION_COOKIE, refreshResult.sealedSession);
        return refreshResult.session;
      }

      await handleSessionRefresh(UNKEY_SESSION_COOKIE, null, refreshResult.reason);
      return null;
      
    } catch (error) {
      console.error('Session refresh error:', {
        error: error instanceof Error ? error.message : 'Unknown error',
        token: token.substring(0, 10) + '...'
      });
      throw new Error("Session refresh error");
    }
  }


  public async createTenant(params: { name: string, userId: string }): Promise<string> {
    const { userId, name } = params;
    if (!name || !userId) throw new Error('Organization/Workspace name and userId are required.')

    const { orgId } = await this.createOrg(name);

    const membership = await WorkOSAuthProvider.provider.userManagement.createOrganizationMembership({
      organizationId: orgId,
      userId,
      roleSlug: "admin"
    });
    
    // Refresh session with new organization context
    await this.refreshSession(membership.organizationId);

    // return the orgId back to use as the workspace tenant
    return membership.organizationId;
  }

  protected async getOrgId(): Promise<string | null> {
    try {
      const authSession = await this.validateSession();
      if (!authSession) {
        // redirect? middleware should catch before this function
        throw new Error("No auth session.");
      }

      const { orgId } = authSession;
      return orgId;

    } catch (error) {
      throw new Error("Couldn't get orgId.")
    }
  }

  async getCurrentUser(): Promise<any | null> {
    try {
      // Extract the user data from the session cookie
      // Return the UNKEY user shape
      const token = await this.getSession();
      if (!token) return null;

      const WORKOS_COOKIE_PASSWORD = env().WORKOS_COOKIE_PASSWORD;
      if (!WORKOS_COOKIE_PASSWORD) {
        throw new Error("WORKOS_COOKIE_PASSWORD is required");
      }

      try {
        const session = WorkOSAuthProvider.provider.userManagement.loadSealedSession({
          sessionData: token,
          cookiePassword: WORKOS_COOKIE_PASSWORD
        });

        const authResult = await session.authenticate();
        if (authResult.authenticated) {

          const {user, organizationId} = authResult;

          return {
            userId: user.id,
            orgId: organizationId,
	          email: user.email,
	          firstName: user.firstName,
	          lastName: user.lastName,
	          avatarUrl: user.profilePictureUrl,
          }
        }

        else {
          console.error("Get current user failed:", authResult.reason)
          return null;
        }
        
      } catch (error) {
        console.error("Error validating session:", error)
        return null;
      }
    } catch (error) {
      console.error('Error getting user:', error);
      return null;
    }
  }

  async listMemberships(): Promise<any[]> {
    throw new Error("Method not implemented.");
  }

  async signUpViaEmail(email: string): Promise<MagicAuth> {
    if (!email) {
      throw new Error("No email address provided.");
    }
    return WorkOSAuthProvider.provider.userManagement.createMagicAuth({ email });
  }

  async signIn(orgId?: string): Promise<void> {
    throw new Error("Method not implemented.");
  }

  signInViaOAuth({ 
    redirectUrl = env().NEXT_PUBLIC_WORKOS_REDIRECT_URI, 
    provider,
    redirectUrlComplete = SIGN_IN_REDIRECT
  }: SignInViaOAuthOptions): string {
    if (!provider) {
      throw new Error('Provider is required');
    }

    const state = encodeURIComponent(JSON.stringify({ redirectUrlComplete }));

    return WorkOSAuthProvider.provider.userManagement.getAuthorizationUrl({
      clientId: WorkOSAuthProvider.clientId,
      redirectUri: redirectUrl,
      provider: provider === "github" ? "GitHubOAuth" : "GoogleOAuth",
      state
    });
  }

  async completeOAuthSignIn(callbackRequest: Request): Promise<OAuthResult> {
    const url = new URL(callbackRequest.url);
    const code = url.searchParams.get('code');
    const state = url.searchParams.get('state');
    const redirectUrlComplete = state 
      ? JSON.parse(decodeURIComponent(state)).redirectUrlComplete 
      : SIGN_IN_REDIRECT;
  
    if (!code) {
      return {
        success: false,
        redirectTo: SIGN_IN_URL,
        cookies: [],
        error: new Error("No code provided")
      };
    }
  
    try {
      const { sealedSession } = await WorkOSAuthProvider.provider.userManagement.authenticateWithCode({
        clientId: WorkOSAuthProvider.clientId,
        code,
        session: {
          sealSession: true,
          cookiePassword: env().WORKOS_COOKIE_PASSWORD
        }
      });
  
      if (!sealedSession) {
        throw new Error('No sealed session returned from WorkOS');
      }
  
      return {
        success: true,
        redirectTo: redirectUrlComplete,
        cookies: [{
          name: UNKEY_SESSION_COOKIE,
          value: sealedSession,
          options: {
            secure: true,
            httpOnly: true,
          }
        }]
      };
    } catch (error) {
      console.error("OAuth callback failed", error);
      return {
        success: false,
        redirectTo: SIGN_IN_URL,
        cookies: [],
        error: error instanceof Error ? error : new Error('Unknown error')
      };
    }
  }

  async signOut(): Promise<string | null> {
    const token = await this.getSession();
    if (!token) {
      console.error('Session cookie not found');
      return null;
    }

    const WORKOS_COOKIE_PASSWORD = env().WORKOS_COOKIE_PASSWORD;
    if (!WORKOS_COOKIE_PASSWORD) {
      throw new Error("WORKOS_COOKIE_PASSWORD is required");
    }

    try {
      const session = WorkOSAuthProvider.provider.userManagement.loadSealedSession({
        sessionData: token,
        cookiePassword: WORKOS_COOKIE_PASSWORD
      });

      return await session.getLogoutUrl();
    }

    catch (error) {
      console.error('WorkOS Session error:', {
        error: error instanceof Error ? error.message : 'Unknown error',
        token: token.substring(0, 10) + '...'
      });
      return null;
    }
  }

  async updateTenant(org: Partial<any>): Promise<any> {
    throw new Error("Method not implemented.");
  }

  protected async createOrg(name: string): Promise<Organization> {
    if (!name) {
      throw new Error("Organization/workspace name is required.")
    }

    const org = await WorkOSAuthProvider.provider.organizations.createOrganization({ name });

    return {
      orgId: org.id,
      name
    };

  }
}