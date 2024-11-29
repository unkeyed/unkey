import { type MagicAuth, WorkOS } from "@workos-inc/node";
import { AuthSession, BaseAuthProvider, UNKEY_SESSION_COOKIE, type SignInViaOAuthOptions } from "./interface";
import { NextRequest, NextResponse } from "next/server";
import { env } from "@/lib/env";

const SSO_CALLBACK_URI = "/auth/sso-callback";
const SIGN_IN_REDIRECT = "/apis";
const SIGN_UP_REDIRECT = "/new";
const SIGN_IN_URL = "/auth/sign-in";

interface OAuthResult {
  success: boolean;
  error?: any;
  redirectTo: string;
  cookies: Array<{
    name: string;
    value: string;
    options: {
      secure?: boolean;
      httpOnly?: boolean;
      sameSite?: 'lax' | 'strict' | 'none';
      path?: string;
    };
  }>;
}

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
    WorkOSAuthProvider.provider = new WorkOS(config.apiKey, {clientId: config.clientId});
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
          orgId: organizationId || '' // if they are a brand new user and they haven't hit the workspace creation flow, they won't have an orgId
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

  // WIP
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

  public signInViaOAuth({ 
    redirectUrl = env().NEXT_PUBLIC_WORKOS_REDIRECT_URI, 
    provider,
    redirectUrlComplete = SIGN_IN_REDIRECT
  }: SignInViaOAuthOptions): String {
      if (!provider) {
        throw new Error('Provider is required');
      }

      // add the redirect as state to access it in the callback later
      const state = encodeURIComponent(JSON.stringify({
        redirectUrlComplete
      }));

      const authorizationUrl = WorkOSAuthProvider.provider.userManagement.getAuthorizationUrl({
        clientId: WorkOSAuthProvider.clientId,
        redirectUri: redirectUrl,
        provider: provider === "github" ? "GitHubOAuth" : "GoogleOAuth",
        state
      });
      
      return authorizationUrl;
  }

  public async completeOAuthSignIn(callbackRequest: NextRequest): Promise<OAuthResult> {

    const searchParams = callbackRequest.nextUrl.searchParams;
    const code = searchParams.get('code');
    const state = searchParams.get('state');

    const redirectUrlComplete = state 
    ? JSON.parse(decodeURIComponent(state)).redirectUrlComplete 
    : SIGN_IN_REDIRECT; // state *shouldn't* be null, but just in case

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

      // make Typescript happy because it can be undefined
      // but only the session property from `authenticateWithCode` is not included
      // we always need the session to come back, so if it doesn't, don't set a session cookie
      if (!sealedSession) {
        throw new Error('No sealed session returned from WorkOS');
      }

      // TODO: make cookies a single object? Originally was setting a user cookie
      // but userId/orgId can be accessed from the session by unsealing it
      // caveat: can only be unsealed server-side
      return {
        success: true,
        redirectTo: redirectUrlComplete,
        cookies: [
          {
            name: UNKEY_SESSION_COOKIE,
            value: sealedSession,
            options: {
              secure: true,
              httpOnly: true,
            }
          }
        ]
      };

    }
    catch (error) {
      console.error("Callback failed", error);
      return {
        success: false,
        redirectTo: '/auth/sign-in',
        cookies: [],
        error
      };
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
