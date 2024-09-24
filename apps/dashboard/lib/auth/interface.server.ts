export type ServerUser = {
  id: string;
};

export interface ServerAuth {
  // import { serverAuth } from "@/lib/auth" must return the tenant ID of the current workspace.
  // If there is none, it must trigger a redirect to the sign in page.
  getTenantId(): Promise<string>;

  getUser(): Promise<ServerUser | null>;

  getUserFromCookie(cookie: string): Promise<ServerUser | null>
}
