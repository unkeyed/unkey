import {
  AuthErrorCode,
  type AuthErrorResponse,
  type EmailAuthResult,
  type Invitation,
  type InvitationListResponse,
  type Membership,
  type MembershipListResponse,
  type NavigationResponse,
  type OAuthResult,
  type OrgInviteParams,
  type Organization,
  type SessionRefreshResult,
  type SessionValidationResult,
  type SignInViaOAuthOptions,
  type StateChangeResponse,
  type UpdateMembershipParams,
  type UpdateOrgParams,
  type User,
  type UserData,
  type VerificationResult,
  errorMessages,
} from "./types";

/**
 * BaseAuthProvider
 *
 * Abstract class defining the interface for authentication providers.
 * Implementations of this class handle user authentication, session management,
 * organization/tenant management, and user management operations.
 */
export abstract class BaseAuthProvider {
  /**
   * Validates a session token and returns information about its validity.
   *
   * @param sessionToken - The session token to validate
   * @returns Information about the session including validity, user ID, and organization
   */
  abstract validateSession(sessionToken: string): Promise<SessionValidationResult>;

  /**
   * Refreshes an existing session token and returns a new token.
   *
   * @param sessionToken - The session token to refresh
   * @returns A new session token and related session information
   * @throws Error if the session cannot be refreshed
   */
  abstract refreshSession(sessionToken: string): Promise<SessionRefreshResult>;

  /**
   * Initiates an email-based sign-in process for the specified email.
   *
   * @param params - Parameters containing email and optional request metadata
   * @returns Result of the sign-in attempt
   */
  abstract signInViaEmail(params: {
    email: string;
    ipAddress?: string;
    userAgent?: string;
    bypassRadar?: boolean;
  }): Promise<EmailAuthResult>;

  /**
   * Verifies an authentication code sent to a user's email.
   *
   * @param params - Parameters containing the email, verification code, and optional invitation token
   * @returns Result of the verification process, including redirect information on success
   */
  abstract verifyAuthCode(params: {
    email: string;
    code: string;
    invitationToken?: string;
  }): Promise<VerificationResult>;

  /**
   * Verifies a user's email address using a verification code.
   *
   * @param params - Parameters containing the verification code and token
   * @returns Result of the email verification process
   */
  abstract verifyEmail(params: {
    code: string;
    token: string;
  }): Promise<VerificationResult>;

  /**
   * Resends an authentication code to the specified email address.
   *
   * @param email - The email address to resend the auth code to
   * @returns Result of the resend attempt
   */
  abstract resendAuthCode(email: string): Promise<EmailAuthResult>;

  /**
   * Creates a new user account with the provided user data.
   *
   * @param params - User data including email, first name, last name, and optional request metadata
   * @returns Result of the sign-up attempt
   */
  abstract signUpViaEmail(
    params: UserData & {
      ipAddress?: string;
      userAgent?: string;
      bypassRadar?: boolean;
    },
  ): Promise<EmailAuthResult>;

  /**
   * Gets the URL to redirect users to for signing out.
   *
   * @returns URL for sign-out or null if not applicable
   */
  abstract getSignOutUrl(): Promise<string | null>;

  /**
   * Completes the organization selection process during authentication.
   *
   * @param params - Parameters containing the selected organization ID and pending auth token
   * @returns Result of the organization selection process
   */
  abstract completeOrgSelection(params: {
    orgId: string;
    pendingAuthToken: string;
  }): Promise<VerificationResult>;

  /**
   * Initiates OAuth-based authentication with the specified provider.
   *
   * @param options - OAuth configuration including provider type and redirect URL
   * @returns The URL to redirect the user to for OAuth authentication
   */
  abstract signInViaOAuth(options: SignInViaOAuthOptions): string;

  /**
   * Completes the OAuth sign-in process after the user is redirected back.
   *
   * @param callbackRequest - The request object from the OAuth provider callback
   * @returns Result of the OAuth authentication process
   */
  abstract completeOAuthSignIn(callbackRequest: Request): Promise<OAuthResult>;

  /**
   * Retrieves a user by their unique ID.
   *
   * @param userId - The ID of the user to retrieve
   * @returns The user object if found, null otherwise
   */
  abstract getUser(userId: string): Promise<User | null>;

  /**
   * Finds a user by their email address.
   *
   * @param email - The email address to search for
   * @returns The user object if found, null otherwise
   */
  abstract findUser(email: string): Promise<User | null>;

  /**
   * Creates a new tenant (organization) and associates it with a user.
   *
   * @param params - Parameters containing organization name and user ID
   * @returns The ID of the newly created organization
   */
  abstract createTenant(params: {
    name: string;
    userId: string;
  }): Promise<string>;

  /**
   * Updates an existing organization's information.
   *
   * @param params - Parameters containing the organization ID and new details
   * @returns The updated organization object
   */
  abstract updateOrg(params: UpdateOrgParams): Promise<Organization>;

  /**
   * Creates a new organization with the specified name.
   * Protected method intended for internal use by provider implementations.
   *
   * @param name - The name of the organization to create
   * @returns The newly created organization object
   */
  protected abstract createOrg(name: string): Promise<Organization>;

  /**
   * Retrieves an organization by its unique ID.
   *
   * @param orgId - The ID of the organization to retrieve
   * @returns The organization object
   * @throws Error if the organization is not found
   */
  abstract getOrg(orgId: string): Promise<Organization>;

  /**
   * Switches the current user's active organization.
   *
   * @param newOrgId - The ID of the organization to switch to
   * @returns A new session with the updated organization context
   */
  abstract switchOrg(newOrgId: string): Promise<SessionRefreshResult>;

  /**
   * Lists all memberships for a specific user.
   *
   * @param userId - The ID of the user to list memberships for
   * @returns List of memberships and metadata
   */
  abstract listMemberships(userId: string): Promise<MembershipListResponse>;

  /**
   * Lists all members of a specific organization.
   *
   * @param orgId - The ID of the organization to list members for
   * @returns List of memberships and metadata
   */
  abstract getOrganizationMemberList(orgId: string): Promise<MembershipListResponse>;

  /**
   * Updates a user's membership role within an organization.
   *
   * @param params - Parameters containing the membership ID and new role
   * @returns The updated membership object
   */
  abstract updateMembership(params: UpdateMembershipParams): Promise<Membership>;

  /**
   * Removes a user's membership from an organization.
   *
   * @param membershipId - The ID of the membership to remove
   * @returns A promise that resolves when the membership is removed
   */
  abstract removeMembership(membershipId: string): Promise<void>;

  /**
   * Deactivates a user's membership in an organization (soft delete).
   *
   * @param membershipId - The ID of the membership to deactivate
   * @returns The deactivated membership object
   */
  abstract deactivateMembership(membershipId: string): Promise<Membership>;

  /**
   * Invites a new user to join an organization.
   *
   * @param params - Parameters containing the organization ID, email, and role
   * @returns The created invitation object
   */
  abstract inviteMember(params: OrgInviteParams): Promise<Invitation>;

  /**
   * Lists all pending invitations for an organization.
   *
   * @param orgId - The ID of the organization to list invitations for
   * @returns List of invitations and metadata
   */
  abstract getInvitationList(orgId: string): Promise<InvitationListResponse>;

  /**
   * Retrieves an invitation by its token.
   *
   * @param invitationToken - The token of the invitation to retrieve
   * @returns The invitation object if found, null otherwise
   */
  abstract getInvitation(invitationToken: string): Promise<Invitation | null>;

  /**
   * Revokes an existing invitation.
   *
   * @param invitationId - The ID of the invitation to revoke
   * @returns A promise that resolves when the invitation is revoked
   */
  abstract revokeOrgInvitation(invitationId: string): Promise<void>;

  /**
   * Accepts an invitation to join an organization.
   *
   * @param invitationId - The ID of the invitation to accept
   * @returns The updated invitation object
   */
  abstract acceptInvitation(invitationId: string): Promise<Invitation>;

  /**
   * Standardized error handling for authentication operations.
   * Converts various error types into a consistent format.
   *
   * @param error - The error to handle
   * @returns A standardized error response
   */
  protected handleError(error: unknown): AuthErrorResponse {
    if (error instanceof Error) {
      // Handle provider-specific errors
      if ("message" in error && typeof error.message === "string") {
        const errorCode = error.message as AuthErrorCode;
        if (errorCode in AuthErrorCode) {
          return {
            success: false,
            code: errorCode,
            message: errorMessages[errorCode],
          };
        }
      }

      // Handle generic errors
      return {
        success: false,
        code: AuthErrorCode.UNKNOWN_ERROR,
        message: error.message,
      };
    }

    // Fallback error
    return {
      success: false,
      code: AuthErrorCode.UNKNOWN_ERROR,
      message: errorMessages[AuthErrorCode.UNKNOWN_ERROR],
    };
  }

  /**
   * Creates a standard response for operations that change state.
   *
   * @returns A success response
   */
  protected createStateChangeResponse(): StateChangeResponse {
    return { success: true };
  }

  /**
   * Creates a standard response for operations that require navigation.
   *
   * @param redirectTo - The URL to redirect to
   * @param cookies - Cookies to set in the response
   * @returns A navigation response with redirect information
   */
  protected createNavigationResponse(
    redirectTo: string,
    cookies: NavigationResponse["cookies"],
  ): NavigationResponse {
    return {
      success: true,
      redirectTo,
      cookies,
    };
  }
}
