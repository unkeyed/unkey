import { type MagicAuth, WorkOS } from "@workos-inc/node";
import { AuthSession, OAuthResult, Organization, OrgMembership, Membership, UNKEY_SESSION_COOKIE, User, type SignInViaOAuthOptions, UpdateOrgParams, OrgInvite, Invitation, OrgInvitation, UpdateMembershipParams } from "./types";
import { env } from "@/lib/env";
import { getCookie, updateCookie } from "./cookies";
import { BaseAuthProvider } from "./base-provider"

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

    const { id } = await this.createOrg(name);

    const membership = await WorkOSAuthProvider.provider.userManagement.createOrganizationMembership({
      organizationId: id,
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
        id: organization.id,
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

  async getUser(userId: string): Promise<User | null> {
    if (!userId) {
      throw new Error("User Id is required.");
    }

    try {
      const user = await WorkOSAuthProvider.provider.userManagement.getUser(userId);
      if (!user) {
        return null;
      }

      return {
        id: user.id,
        orgId: null, // not available from the user management user lookup
        email: user.email,
        firstName: user.firstName,
        lastName: user.lastName,
        fullName: user.firstName + " " + user.lastName,
        avatarUrl: user.profilePictureUrl,
      }
    } catch (error) {
      console.error("Failed to get user: ", error);
      return null;
    }
  }

  async listMemberships(): Promise<OrgMembership> {
      const user = await this.getCurrentUser();
      if (!user) return {
        data: [],
        metadata: {}
      };
      const { id: userId } = user;

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
      orgs.map(org => [org.id, org.name])
    );

    return {
      data: memberships.data.map((membership) => {
      
        return {
          id: membership.id,
          user,
          organization: {
            name: orgMap.get(membership.organizationId) || "Unknown Organization",
            id: membership.organizationId
          },
          role: membership.role.slug,
          createdAt: membership.createdAt,
          updatedAt: membership.updatedAt,
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
  
  async updateOrg({ id, name }: UpdateOrgParams): Promise<Organization> {
    if (!id) {
      throw new Error("Organization id is required.");
    }
  
    if (!name) {
      throw new Error("Organization name is required.");
    }
  
    try {
      const updatedOrg = await WorkOSAuthProvider.provider.organizations.updateOrganization({
        organization: id,
        name
      });
  
      return {
        id: updatedOrg.id,
        name: updatedOrg.name,
        createdAt: updatedOrg.createdAt,
        updatedAt: updatedOrg.updatedAt
      };
    } catch (error) {
      console.error("Failed to update organization:", error);
      throw new Error(
        error instanceof Error 
          ? error.message 
          : "Failed to update organization"
      );
    }
  }

  protected async createOrg(name: string): Promise<Organization> {
    if (!name) {
      throw new Error("Organization/workspace name is required.")
    }

    const org = await WorkOSAuthProvider.provider.organizations.createOrganization({ name });

    return {
      id: org.id,
      name: org.name,
      createdAt: org.createdAt,
      updatedAt: org.updatedAt
    };

  }

  async getOrganizationMemberList(orgId: string): Promise<OrgMembership> {
    if (!orgId) {
        throw new Error("Organization id is required.");
    }

    try {
        // Get the organization info
        const org = await this.getOrg(orgId);
        
        // Get all members of the organization
        const members = await WorkOSAuthProvider.provider.userManagement.listOrganizationMemberships({
            organizationId: orgId,
            limit: 100,
            statuses: ["active"]
        });

        // Get user data for each member
        const userPromises = members.data.map(member => 
            this.getUser(member.userId).catch((error: any) => {
                console.error(`Failed to fetch user ${member.userId}:`, error);
                return null;
            })
        );
        const users = await Promise.all(userPromises);

        // Create user lookup map, filtering out failed fetches
        const usersMap = new Map<string, User>(
            users.filter((user): user is User => user !== null)
                .map(user => [user.id, user])
        );

        return {
            data: members.data.map((member): Membership => {
                const user = usersMap.get(member.userId);

                if (!user) {
                    throw new Error(`User ${member.userId} not found`);
                }

                return {
                    id: member.id,
                    user,
                    organization: org,
                    role: member.role.slug,
                    createdAt: member.createdAt,
                    updatedAt: member.updatedAt,
                    status: member.status
                };
            }),
            metadata: members.listMetadata ?? {}
        };
    } catch (error) {
        console.error("Failed to get organization member list:", error);
        return {
            data: [],
            metadata: {}
        };
    }
  }

  async inviteMember({ orgId, email, role = "basic_member" }: OrgInvite): Promise<Invitation> {
    if (!orgId) {
      throw new Error("Organization id is required.");
    }
  
    if (!email) {
      throw new Error("Recipient email is required.");
    }
  
    try {
      const user = await this.getCurrentUser();
      if (!user) {
        throw new Error("User must be authenticated to invite members.");
      }
  
      const invitation = await WorkOSAuthProvider.provider.userManagement.sendInvitation({
        email,
        organizationId: orgId,
        roleSlug: role,
        inviterUserId: user.id
      });
  
      return {
        id: invitation.id,
        email: invitation.email,
        state: invitation.state,
        acceptedAt: invitation.acceptedAt,
        revokedAt: invitation.revokedAt,
        expiresAt: invitation.expiresAt,
        token: invitation.token,
        organizationId: invitation.organizationId || orgId,
        inviterUserId: invitation.inviterUserId || user.id,
        createdAt: invitation.createdAt,
        updatedAt: invitation.updatedAt,
      };
  
    } catch (error) {
      console.error('Failed to send invitation:', error);
      throw error instanceof Error 
        ? error 
        : new Error('Failed to send invitation');
    }
  }

  async updateMembership({membershipId, role}: UpdateMembershipParams): Promise<Membership> {
    if (!membershipId) {
      throw new Error("Membership id is required.");
    }

    if (!role) {
      throw new Error("Role is required");
    }

    try {
      const membership = await WorkOSAuthProvider.provider.userManagement.updateOrganizationMembership(
        membershipId,
        {roleSlug: role}
      );

      const org = await this.getOrg(membership.organizationId);
      const user = await this.getUser(membership.userId);

      return {
        id: membership.id,
        user: user!,
        organization: org,
        role: membership.role.slug,
        createdAt: membership.createdAt,
        updatedAt: membership.updatedAt,
        status: membership.status
      };
    }
    catch (error) {
      console.error('Failed to update membership:', error);
      throw error instanceof Error 
        ? error 
        : new Error('Failed to update membership');
    }
  }

  async removeMembership(membershipId: string): Promise<void> {
    if (!membershipId) {
      throw new Error("Membership Id is required");
    }

    try {
      await WorkOSAuthProvider.provider.userManagement.deleteOrganizationMembership(membershipId);
    }

    catch(error) {
      console.error("Failed to delete membership: ", error);
    }
  } 

  async revokeOrgInvitation(invitationId: string): Promise<void> {
    if (!invitationId) {
      throw new Error("Membership Id is required");
    }

    try {
      await WorkOSAuthProvider.provider.userManagement.revokeInvitation(invitationId);
    }

    catch(error) {
      console.error("Failed to revoke invitation: ", error);
    }
  } 

  async getInvitationList(orgId: string): Promise<OrgInvitation> {
    if (!orgId) {
      throw new Error("Organization Id is required");
    }

    try {
      const invitationsList = await WorkOSAuthProvider.provider.userManagement.listInvitations({
        organizationId: orgId
      });

      return {
        data: invitationsList.data.map((invitation): Invitation => {
          return {
            id: invitation.id,
            email: invitation.email,
            state: invitation.state,
            acceptedAt: invitation.acceptedAt,
            revokedAt: invitation.revokedAt,
            expiresAt: invitation.expiresAt,
            token: invitation.token,
            organizationId: invitation.organizationId ?? orgId,
            inviterUserId: invitation.inviterUserId ?? undefined,
            createdAt: invitation.createdAt,
            updatedAt: invitation.updatedAt,
          }
        }),
        metadata: invitationsList.listMetadata
      }
      
    } catch (error) {
      console.error("Failed to get organization invitations list:", error);
        return {
            data: [],
            metadata: {}
        };
    }
  }
}