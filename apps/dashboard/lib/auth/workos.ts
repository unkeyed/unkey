import type { Auth } from './interface';

export class WorkosAuth<T> implements Auth<T> {
    constructor() {
      // Initialize any necessary properties or services
    }
  
    async getOrgId(): Promise<T> {
      // Implementation to get the organization ID
      // If none, trigger a redirect to the sign-in page
      throw new Error("Method not implemented.");
    }
  
    async getSession(): Promise<{ userId: string; orgId: string } | null> {
      // Implementation to get the session
      throw new Error("Method not implemented.");
    }
  
    async getUser(): Promise<{ userId: string; profileUrl: string; name: string } | null> {
      // Implementation to get the user data
      throw new Error("Method not implemented.");
    }
  
    async listOrganisations(): Promise<T> {
      // Implementation to list organizations
      throw new Error("Method not implemented.");
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