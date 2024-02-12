import { Workspace, db, schema } from "@/lib/db";
import { WorkOS as Client } from "@workos-inc/node";
import { Auth } from "./interface";

export class WorkOS implements Auth {
  private readonly client: Client;
  private readonly clientId: string;

  constructor(opts: {
    clientId: string;
    apiKey: string;
  }) {
    this.clientId = opts.clientId;
    this.client = new Client(opts.apiKey);
  }

  public async listWorkspaces(userId: string): Promise<Workspace[]> {
    const memberships = await this.client.userManagement.listOrganizationMemberships({
      userId,
    });
    console.log({ memberships });

    const workspaces = await db.query.workspaces.findMany({
      where: (table, { inArray }) =>
        inArray(
          table.tenantId,
          memberships.data.map((m) => m.organizationId),
        ),
    });
    return workspaces;
  }

  public async createWorkspace(
    userId: string,
    workspace: Omit<Workspace, "tenantId">,
  ): Promise<void> {
    const org = await this.client.organizations.createOrganization({
      name: workspace.name,
    });
    await db.insert(schema.workspaces).values({
      ...workspace,
      tenantId: org.id,
    });
    await this.client.userManagement.createOrganizationMembership({
      userId,
      organizationId: org.id,
    });
  }
}
