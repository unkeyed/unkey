export type ServerUser = {
  id: string;
  tenantId: string;
};

export type Organisation = {
  // id is going to be used as `tenantId` in our workspace table
  id: string;
  name: string;
  imageUrl: string;
};

export interface ServerAuth {
  // import { serverAuth } from "@/lib/auth" must return the tenant ID of the current workspace.
  // If there is none, it must trigger a redirect to the sign in page.
  getTenantId(): Promise<string>;

  getUser(): Promise<ServerUser | null>;

  listOrganisations(): Promise<Organisation[]>;

  // signIn(orgId?: string): Promise<void>;

  // signOut(): Promise<void>;

  // updateOrg(org: Partial<Organisation>): Promise<void>;
}
