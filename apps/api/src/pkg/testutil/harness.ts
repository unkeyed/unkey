import { Client } from "@planetscale/database";
import { ClickHouse } from "@unkey/clickhouse";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import type { TaskContext } from "vitest";
import {
  type Api,
  type Database,
  type InsertPermission,
  type KeyAuth,
  type Permission,
  type Role,
  type Workspace,
  drizzle,
  eq,
  schema,
} from "../db";
import { databaseEnv } from "./env";

export type Resources = {
  unkeyWorkspace: Workspace;
  unkeyApi: Api;
  unkeyKeyAuth: KeyAuth;
  userWorkspace: Workspace;
  userApi: Api;
  userKeyAuth: KeyAuth;
};

export abstract class Harness {
  public readonly db: { primary: Database; readonly: Database };
  public readonly ch: ClickHouse;
  public resources: Resources;

  constructor(t: TaskContext) {
    const { DATABASE_HOST, DATABASE_PASSWORD, DATABASE_USERNAME, CLICKHOUSE_URL } =
      databaseEnv.parse(process.env);

    const db = drizzle(
      new Client({
        host: DATABASE_HOST,
        username: DATABASE_USERNAME,
        password: DATABASE_PASSWORD,
        fetch: (url, init) => {
          const u = new URL(url);
          if (u.hostname === "planetscale" || u.host.includes("localhost")) {
            u.protocol = "http";
          }
          return fetch(u, init);
        },
      }),
      {
        schema,
      },
    );

    this.db = { primary: db, readonly: db };
    this.ch = new ClickHouse({ url: CLICKHOUSE_URL });
    this.resources = this.createResources();

    t.onTestFinished(async () => {
      await this.teardown();
    });
  }

  private async teardown(): Promise<void> {
    const deleteWorkspaces = async () => {
      for (const workspaceId of [
        this.resources.userWorkspace.id,
        this.resources.unkeyWorkspace.id,
      ]) {
        await this.db.primary
          .delete(schema.workspaces)
          .where(eq(schema.workspaces.id, workspaceId));
      }
    };
    for (let i = 1; i <= 5; i++) {
      try {
        await deleteWorkspaces();
        return;
      } catch (err) {
        if (i === 5) {
          throw err;
        }
        await new Promise((r) => setTimeout(r, i * 500));
      }
    }
  }
  /**
   * Create a new root key with optional roles
   */
  async createRootKey(permissions?: string[]) {
    const rootKey = new KeyV1({ byteLength: 16, prefix: "unkey" }).toString();
    const start = rootKey.slice(0, 10);
    const keyId = newId("test");
    const hash = await sha256(rootKey);

    await this.db.primary.insert(schema.keys).values({
      id: keyId,
      keyAuthId: this.resources.unkeyKeyAuth.id,
      hash,
      start,
      workspaceId: this.resources.unkeyWorkspace.id,
      forWorkspaceId: this.resources.userWorkspace.id,
      createdAtM: Date.now(),
    });
    if (permissions && permissions.length > 0) {
      const create: InsertPermission[] = permissions.map((name) => ({
        id: newId("test"),
        name,
        slug: name,
        description: null,
        workspaceId: this.resources.unkeyWorkspace.id,
        updatedAtM: null,
        createdAtM: Date.now(),
        deletedAtM: null,
      }));

      await this.db.primary.insert(schema.permissions).values(create);
      await this.db.primary.insert(schema.keysPermissions).values(
        create.map((p) => ({
          keyId,
          permissionId: p.id,
          workspaceId: this.resources.unkeyWorkspace.id,
        })),
      );
    }
    return {
      id: keyId,
      key: rootKey,
    };
  }

  public async createKey(opts?: {
    identityId?: string;
    roles?: {
      name: string;
      permissions?: string[];
    }[];
  }): Promise<{
    keyId: string;
    key: string;
    identityId?: string;
  }> {
    /**
     * Prepare the key we'll use
     */
    const key = new KeyV1({ prefix: "test", byteLength: 32 }).toString();
    const hash = await sha256(key);
    const keyId = newId("test");
    await this.db.primary.insert(schema.keys).values({
      id: keyId,
      identityId: opts?.identityId,
      keyAuthId: this.resources.userKeyAuth.id,
      hash,
      start: key.slice(0, 8),
      workspaceId: this.resources.userWorkspace.id,
      createdAtM: Date.now(),
    });

    for (const role of opts?.roles ?? []) {
      const { id: roleId } = await this.optimisticUpsertRole(
        this.resources.userWorkspace.id,
        role.name,
      );
      await this.db.primary.insert(schema.keysRoles).values({
        keyId,
        roleId,
        workspaceId: this.resources.userWorkspace.id,
      });

      for (const permissionName of role.permissions ?? []) {
        const permission = await this.optimisticUpsertPermission(
          this.resources.userWorkspace.id,
          permissionName,
        );
        await this.db.primary
          .insert(schema.rolesPermissions)
          .values({
            roleId,
            permissionId: permission.id,
            workspaceId: this.resources.userWorkspace.id,
          })
          .onDuplicateKeyUpdate({
            set: {
              updatedAtM: Date.now(),
            },
          })
          .catch((err) => {
            console.error(JSON.stringify(err), err);
            throw err;
          });
      }
    }

    return {
      keyId,
      key,
      identityId: opts?.identityId,
    };
  }

  private async optimisticUpsertPermission(workspaceId: string, name: string): Promise<Permission> {
    const permission: Permission = {
      id: newId("test"),
      name,
      slug: name,
      workspaceId,
      createdAtM: Date.now(),
      updatedAtM: null,
      description: null,
    };

    return this.db.primary.transaction(async (tx) => {
      const found = await tx.query.permissions.findFirst({
        where: (table, { and, eq }) =>
          and(eq(table.workspaceId, workspaceId), eq(table.name, name)),
      });
      if (found) {
        return found;
      }

      await tx.insert(schema.permissions).values(permission);

      return permission;
    });
  }

  private async optimisticUpsertRole(workspaceId: string, name: string): Promise<Role> {
    const role: Role = {
      id: newId("test"),
      name,
      workspaceId,
      createdAtM: Date.now(),
      updatedAtM: null,
      description: null,
    };
    return this.db.primary.transaction(async (tx) => {
      const found = await tx.query.roles.findFirst({
        where: (table, { and, eq }) =>
          and(eq(table.workspaceId, workspaceId), eq(table.name, name)),
      });
      if (found) {
        return found;
      }

      await tx.insert(schema.roles).values(role);

      return role;
    });
  }

  public createResources(): Resources {
    const unkeyWorkspace: Workspace = {
      id: newId("test"),
      name: "unkey",
      slug: newId("test"),
      orgId: newId("test"),
      plan: "enterprise",
      tier: "Enterprise",
      features: {},
      betaFeatures: {},
      stripeCustomerId: null,
      stripeSubscriptionId: null,
      subscriptions: null,
      createdAtM: Date.now(),
      enabled: true,
      deleteProtection: true,
      updatedAtM: null,
      deletedAtM: null,
      partitionId: null,
      k8sNamespace: null,
    };
    const userWorkspace: Workspace = {
      id: newId("test"),
      name: "user",
      slug: newId("test"),
      orgId: newId("test"),
      plan: "pro",
      tier: "Pro Max",
      features: {},
      betaFeatures: {},
      stripeCustomerId: null,
      stripeSubscriptionId: null,
      subscriptions: null,
      createdAtM: Date.now(),
      enabled: true,
      deleteProtection: true,
      updatedAtM: null,
      deletedAtM: null,
      partitionId: null,
      k8sNamespace: null,
    };

    const unkeyKeyAuth: KeyAuth = {
      id: newId("test"),
      workspaceId: unkeyWorkspace.id,
      createdAtM: Date.now(),
      storeEncryptedKeys: false,
      updatedAtM: null,
      deletedAtM: null,
      defaultPrefix: null,
      defaultBytes: null,
      sizeApprox: 0,
      sizeLastUpdatedAt: 0,
    };
    const userKeyAuth: KeyAuth = {
      id: newId("test"),
      workspaceId: userWorkspace.id,
      createdAtM: Date.now(),
      storeEncryptedKeys: false,
      updatedAtM: null,
      deletedAtM: null,
      defaultPrefix: null,
      defaultBytes: null,
      sizeApprox: 0,
      sizeLastUpdatedAt: 0,
    };

    const unkeyApi: Api = {
      id: newId("test"),
      name: "unkey",
      workspaceId: unkeyWorkspace.id,
      authType: "key",
      keyAuthId: unkeyKeyAuth.id,
      ipWhitelist: null,
      createdAtM: Date.now(),
      deleteProtection: true,
      updatedAtM: null,
      deletedAtM: null,
    };
    const userApi: Api = {
      id: newId("test"),
      name: "user",
      workspaceId: userWorkspace.id,
      authType: "key",
      keyAuthId: userKeyAuth.id,
      ipWhitelist: null,
      createdAtM: Date.now(),
      deleteProtection: true,
      updatedAtM: null,
      deletedAtM: null,
    };

    return {
      unkeyWorkspace,
      unkeyApi,
      unkeyKeyAuth,
      userWorkspace,
      userApi,
      userKeyAuth,
    };
  }

  protected async seed(): Promise<void> {
    await this.db.primary.insert(schema.workspaces).values(this.resources.unkeyWorkspace);
    await this.db.primary.insert(schema.quotas).values({
      workspaceId: this.resources.unkeyWorkspace.id,
      requestsPerMonth: 150_000,
      auditLogsRetentionDays: 30,
      logsRetentionDays: 7,
      team: false,
    });
    await this.db.primary.insert(schema.keyAuth).values(this.resources.unkeyKeyAuth);
    await this.db.primary.insert(schema.apis).values(this.resources.unkeyApi);

    await this.db.primary.insert(schema.workspaces).values(this.resources.userWorkspace);
    await this.db.primary.insert(schema.quotas).values({
      workspaceId: this.resources.userWorkspace.id,
      requestsPerMonth: 150_000,
      auditLogsRetentionDays: 30,
      logsRetentionDays: 7,
      team: false,
    });
    await this.db.primary.insert(schema.keyAuth).values(this.resources.userKeyAuth);
    await this.db.primary.insert(schema.apis).values(this.resources.userApi);
  }
}
