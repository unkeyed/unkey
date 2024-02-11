// type TODO = any

import { Workspace } from "@unkey/db";

export interface Auth {
  listWorkspaces: (userId: string) => Promise<Workspace[]>;
  createWorkspace: (userId: string, workspace: Omit<Workspace, "tenantId">) => Promise<void>;
  // switchWorkspace:(userId: string, workspaceId: string)=> Promise<void>
  // signIn(): Promise<TODO>
  // switchWorkspace(workspaceId: string): Promise<TODO>
}
