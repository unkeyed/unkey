import { type MagicAuth, WorkOS } from "@workos-inc/node";
import { AuthSession, BaseAuthProvider, OAuthResult, Organization, OrgMembership, UNKEY_SESSION_COOKIE, User, type SignInViaOAuthOptions } from "./interface";
import { env } from "@/lib/env";
import { getCookie, updateCookie } from "./cookies";

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

  async validateSession(sessionToken: string): Promise<AuthSession | null> {
    if (!sessionToken) return null;

    const WORKOS_COOKIE_PASSWORD = env().WORKOS_COOKIE_PASSWORD;
    if (!WORKOS_COOKIE_PASSWORD) {
      throw new Error("WORKOS_COOKIE_PASSWORD is required");
    }

    try {
      const session = await WorkOSAuthProvider.provider.userManagement.loadSealedSession({
        sessionData: sessionToken,
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
        token: sessionToken.substring(0, 10) + '...'
      });
      return null;
    }
  }

  async refreshSession(orgId?: string): Promise<void> {
    const token = await getCookie(UNKEY_SESSION_COOKIE);
    if (!token) {
      console.error("No session found");
      return;
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

      const refreshResult = await session.refresh({
        cookiePassword: WORKOS_COOKIE_PASSWORD,
        ...(orgId && { organizationId: orgId })
      });

      if (refreshResult.authenticated) {
        await updateCookie(UNKEY_SESSION_COOKIE, refreshResult.sealedSession);
        //return refreshResult.session;
      }
      else {
        await updateCookie(UNKEY_SESSION_COOKIE, null, refreshResult.reason);
      }
      
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

  protected async getOrg(orgId: string): Promise<Organization> {
    if (!orgId) {
      throw new Error("Organization Id is required.");
    }
    try {
      const organization = await WorkOSAuthProvider.provider.organizations.getOrganization(orgId);
      return {
        orgId: organization.id,
        name: organization.name,
        createdAt: organization.createdAt,
        updatedAt: organization.updatedAt
      }

    } catch (error) {
      throw new Error("Couldn't get organization.")
    }
  }

  async getCurrentUser(): Promise<User | null> {
    try {
      // Extract the user data from the session cookie
      // Return the UNKEY user shape
      const token = await getCookie(UNKEY_SESSION_COOKIE);
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
            id: user.id,
            orgId: organizationId || null,
	          email: user.email,
	          firstName: user.firstName,
	          lastName: user.lastName,
            fullName: user.firstName + " " + user.lastName,
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

  async listMemberships(userId?: string): Promise<OrgMembership> {
    if (!userId) {
      const user = await this.getCurrentUser();
      if (!user) return {
        data: [],
        metadata: {}
      };
      const { id: userId } = user;
    }

    const memberships = await WorkOSAuthProvider.provider.userManagement.listOrganizationMemberships({
      userId,
      limit: 100,
      statuses: ["active"]
    });

    // listOrganizationMembership dhoesn't include orgNames
    const orgPromises = memberships.data.map(membership => this.getOrg(membership.organizationId));
    const orgs = await Promise.all(orgPromises);
  
    //quick org name lookup
    const orgMap = new Map<string, string>(
      orgs.map(org => [org.orgId, org.name])
    );

    return {
      data: memberships.data.map((membership) => {
      
        return {
          id: membership.id,
          orgName: orgMap.get(membership.organizationId) || "Unknown Organization",
          orgId: membership.organizationId,
          role: membership.role.slug,
          createdAt: membership.createdAt,
          status: membership.status
        };
      }),
      metadata: memberships.listMetadata || {}
    };
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

  async getSignOutUrl(): Promise<string | null> {
    const token = await getCookie(UNKEY_SESSION_COOKIE);
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
      name: org.name,
      createdAt: org.createdAt,
      updatedAt: org.updatedAt
    };

  }
}