import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import { ConsoleLogger } from "@unkey/worker-logging";
import type { TaskContext } from "vitest";
import {
  type Api,
  type Database,
  type KeyAuth,
  type Permission,
  type Role,
  type Workspace,
  createConnection,
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
  public resources: Resources;

  constructor(t: TaskContext) {
    const { DATABASE_HOST, DATABASE_PASSWORD, DATABASE_USERNAME } = databaseEnv.parse(process.env);
    const db = createConnection({
      host: DATABASE_HOST,
      username: DATABASE_USERNAME,
      password: DATABASE_PASSWORD,
      retry: 3,
      logger: new ConsoleLogger({ requestId: "" }),
    });
    this.db = { primary: db, readonly: db };
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
      createdAt: new Date(),
    });
    if (permissions && permissions.length > 0) {
      const create: Permission[] = permissions.map((name) => ({
        id: newId("test"),
        name,
        key: name,
        description: null,
        workspaceId: this.resources.unkeyWorkspace.id,
        createdAt: new Date(),
        updatedAt: null,
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
    roles: {
      name: string;
      permissions?: string[];
    }[];
  }): Promise<{ keyId: string; key: string }> {
    /**
     * Prepare the key we'll use
     */
    const key = new KeyV1({ prefix: "test", byteLength: 32 }).toString();
    const hash = await sha256(key);
    const keyId = newId("test");
    await this.db.primary.insert(schema.keys).values({
      id: keyId,
      keyAuthId: this.resources.userKeyAuth.id,
      hash,
      start: key.slice(0, 8),
      workspaceId: this.resources.userWorkspace.id,
      createdAt: new Date(),
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
              updatedAt: new Date(),
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
    };
  }

  private async optimisticUpsertPermission(workspaceId: string, name: string): Promise<Permission> {
    const permission: Permission = {
      id: newId("test"),
      name,
      workspaceId,
      createdAt: new Date(),
      updatedAt: null,
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
      createdAt: new Date(),
      updatedAt: null,
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
      tenantId: newId("test"),
      plan: "enterprise",
      features: {},
      betaFeatures: {},
      stripeCustomerId: null,
      stripeSubscriptionId: null,
      trialEnds: null,
      subscriptions: null,
      planLockedUntil: null,
      planChanged: null,
      createdAt: new Date(),
      deletedAt: null,
      planDowngradeRequest: null,
      enabled: true,
    };
    const userWorkspace: Workspace = {
      id: newId("test"),
      name: "user",
      tenantId: newId("test"),
      plan: "pro",
      features: {},
      betaFeatures: {},
      stripeCustomerId: null,
      stripeSubscriptionId: null,
      trialEnds: null,
      subscriptions: null,
      planLockedUntil: null,
      planChanged: null,
      createdAt: new Date(),
      deletedAt: null,
      planDowngradeRequest: null,
      enabled: true,
    };

    const unkeyKeyAuth: KeyAuth = {
      id: newId("test"),
      workspaceId: unkeyWorkspace.id,
      createdAt: new Date(),
      deletedAt: null,
      storeEncryptedKeys: false,
      createdAtM: Date.now(),
      updatedAtM: null,
      deletedAtM: null,
    };
    const userKeyAuth: KeyAuth = {
      id: newId("test"),
      workspaceId: userWorkspace.id,
      createdAt: new Date(),
      deletedAt: null,
      storeEncryptedKeys: false,
      createdAtM: Date.now(),
      updatedAtM: null,
      deletedAtM: null,
    };

    const unkeyApi: Api = {
      id: newId("test"),
      name: "unkey",
      workspaceId: unkeyWorkspace.id,
      authType: "key",
      keyAuthId: unkeyKeyAuth.id,
      ipWhitelist: null,
      createdAt: new Date(),
      deletedAt: null,
    };
    const userApi: Api = {
      id: newId("test"),
      name: "user",
      workspaceId: userWorkspace.id,
      authType: "key",
      keyAuthId: userKeyAuth.id,
      ipWhitelist: null,
      createdAt: new Date(),
      deletedAt: null,
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
    await this.db.primary.insert(schema.keyAuth).values(this.resources.unkeyKeyAuth);
    await this.db.primary.insert(schema.apis).values(this.resources.unkeyApi);

    await this.db.primary.insert(schema.workspaces).values(this.resources.userWorkspace);
    await this.db.primary.insert(schema.keyAuth).values(this.resources.userKeyAuth);
    await this.db.primary.insert(schema.apis).values(this.resources.userApi);
  }
}
