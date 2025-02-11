// import { BaseAuthProvider } from "./base-provider";
// import type { AuthSession, OAuthResult, OrgMembership, SignInViaOAuthOptions, User } from "./types";

// export class LocalAuthProvider<T> extends BaseAuthProvider {
//   signInViaOAuth(options: SignInViaOAuthOptions): string {
//     throw new Error("Method not implemented.");
//   }
//   validateSession(token: string): Promise<AuthSession | null> {
//     throw new Error("Method not implemented.");
//   }
//   getCurrentUser(): Promise<User | null> {
//     throw new Error("Method not implemented.");
//   }
//   listMemberships(userId?: string): Promise<OrgMembership> {
//     throw new Error("Method not implemented.");
//   }
//   completeOAuthSignIn(callbackRequest: Request): Promise<OAuthResult> {
//     throw new Error("Method not implemented.");
//   }
//   getSignOutUrl(): Promise<any> {
//     throw new Error("Method not implemented.");
//   }
//   updateTenant(org: Partial<any>): Promise<any> {
//     throw new Error("Method not implemented.");
//   }
//   constructor() {
//     // Initialize any necessary properties or services
//     super();
//   }

//   signUpViaEmail(email: string): Promise<any> {
//     throw new Error("Method not implemented.");
//   }

//   async getOrgId(): Promise<T> {
//     // Implementation to get the organization ID
//     // If none, trigger a redirect to the sign-in page
//     throw new Error("Method not implemented.");
//   }

//   async getSession(): Promise<AuthSession | null> {
//     // Implementation to get the session
//     throw new Error("Method not implemented.");
//   }

//   async getUser(): Promise<{ userId: string; profileUrl: string; name: string } | null> {
//     // Implementation to get the user data
//     throw new Error("Method not implemented.");
//   }

//   async listOrganizations(): Promise<T> {
//     // Implementation to list organizations
//     throw new Error("Method not implemented.");
//   }

//   async signIn(orgId?: string): Promise<T> {
//     // Implementation to sign in the user
//     throw new Error("Method not implemented.");
//   }

//   async signOut(): Promise<T> {
//     // Implementation to sign out the user
//     throw new Error("Method not implemented.");
//   }

//   async updateOrg(org: Partial<T>): Promise<T> {
//     // Implementation to update the organization
//     throw new Error("Method not implemented.");
//   }
// }
