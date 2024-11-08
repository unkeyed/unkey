import { type MagicAuth, User, WorkOS } from "@workos-inc/node";
import { getSession, withAuth } from "@workos-inc/authkit-nextjs"
import type { Auth } from "./interface";

type UserDetails = {
  email: string;
  firstName: string;
  lastName: string;
};


export class WorkOSAuth<T> implements Auth<T> {
  private static instance: WorkOSAuth<any> | null = null;
  private static provider: WorkOS;

  constructor(WorkOSApiKey: string) {
    if (WorkOSAuth.instance) {
      return WorkOSAuth.instance;
    }

    WorkOSAuth.provider = new WorkOS(WorkOSApiKey);
    WorkOSAuth.instance = this;
  }

  async getOrgId(): Promise<T> {
    // Implementation to get the organization ID
    // If none, trigger a redirect to the sign-in page
    throw new Error("Method not implemented.");
  }

  async getSession(): Promise<any | null> {
    try {
      const session = await getSession();

      if (!session || !session.user) {
        console.error("User not found")
      }

      console.log("mcs session", session);
      return session;
    }
    catch (error) {
      throw new Error("Something went wrong getting the session");
    }
    //throw new Error("Method not implemented.");
  }

  async getUser(): Promise<any | null> {
    // Implementation to get the user data
    try {
      const { user } = await withAuth();
      if (!user) {
        return;
      }
  
      // Assuming user has these properties or you're transforming the data
      return { user };
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

    const magicAuth = await WorkOSAuth.provider.userManagement.createMagicAuth({ email });

    return magicAuth;
  }

  async signIn(orgId?: string): Promise<T> {
    // Implementation to sign in the user
    throw new Error("Method not implemented.");
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
