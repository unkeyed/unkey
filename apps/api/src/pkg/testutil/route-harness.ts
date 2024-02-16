import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import {
  type Api,
  type Database,
  type KeyAuth,
  Permission,
  type Workspace,
  createConnection,
  eq,
  or,
  schema,
} from "../db";
import { init } from "../global";
import { App, newApp } from "../hono/app";
import { unitTestEnv } from "./env";
import { StepRequest, StepResponse, fetchRoute } from "./request";

export type Resources = {
  unkeyWorkspace: Workspace;
  unkeyApi: Api;
  unkeyKeyAuth: KeyAuth;
  userWorkspace: Workspace;
  userApi: Api;
  userKeyAuth: KeyAuth;
};

export class Harness implements Disposable {
  public readonly db: Database;
  public readonly resources: Resources;
  public readonly app: App;
  private seeded = false;

  constructor() {
    const env = unitTestEnv.parse(process.env);
    this.db = createConnection({
      host: env.DATABASE_HOST,
      username: env.DATABASE_USERNAME,
      password: env.DATABASE_PASSWORD,
    });
    this.app = newApp();
    this.resources = this.createResources();
    // @ts-expect-error
    init({ env });
  }

  async [Symbol.dispose]() {
    const tables = [
      schema.keysPermissions,
      schema.rolesPermissions,
      schema.keysRoles,
      schema.permissions,
      schema.roles,
      schema.auditLogs,
      schema.vercelBindings,
      schema.vercelIntegrations,
      schema.keys,
      schema.keyAuth,
      schema.apis,
    ];
    await this.db.transaction(async (tx) => {
      for (const table of tables) {
        await tx
          .delete(table)
          .where(
            or(
              eq(table.workspaceId, this.resources.userWorkspace.id),
              eq(table.workspaceId, this.resources.unkeyWorkspace.id),
            ),
          );
      }
      // this one is special, where the id is not prefixed
      await tx
        .delete(schema.workspaces)
        .where(
          or(
            eq(schema.workspaces.id, this.resources.userWorkspace.id),
            eq(schema.workspaces.id, this.resources.unkeyWorkspace.id),
          ),
        );
    });
  }

  public useRoutes(...registerFunctions: ((app: App) => any)[]): void {
    registerFunctions.forEach((fn) => fn(this.app));
  }

  async do<TReq, TRes>(req: StepRequest<TReq>): Promise<StepResponse<TRes>> {
    return await fetchRoute<TReq, TRes>(this.app, req);
  }
  async get<TRes>(req: Omit<StepRequest<never>, "method">): Promise<StepResponse<TRes>> {
    return await this.do<never, TRes>({ method: "GET", ...req });
  }
  async post<TReq, TRes>(req: Omit<StepRequest<TReq>, "method">): Promise<StepResponse<TRes>> {
    return await this.do<TReq, TRes>({ method: "POST", ...req });
  }
  async put<TReq, TRes>(req: Omit<StepRequest<TReq>, "method">): Promise<StepResponse<TRes>> {
    return await this.do<TReq, TRes>({ method: "PUT", ...req });
  }
  async delete<TRes>(req: Omit<StepRequest<never>, "method">): Promise<StepResponse<TRes>> {
    return await this.do<never, TRes>({ method: "DELETE", ...req });
  }

  /**
   * Create a new root key with optional roles
   */
  async createRootKey(permissions?: string[]) {
    const rootKey = new KeyV1({ byteLength: 16, prefix: "unkey" }).toString();
    const start = rootKey.slice(0, 10);
    const keyId = newId("key");
    const hash = await sha256(rootKey);

    await this.db.insert(schema.keys).values({
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
        id: newId("permission"),
        name,
        key: name,
        description: null,
        workspaceId: this.resources.unkeyWorkspace.id,
        createdAt: new Date(),
        updatedAt: null,
      }));

      await this.db.insert(schema.permissions).values(create);
      await this.db.insert(schema.keysPermissions).values(
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

  public async createKey(opts: {
    roles: {
      name: string;
      permissions?: string[];
    }[];
  }): Promise<{ keyId: string; key: string }> {
    /**
     * Prepare the key we'll use
     */
    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    const keyId = newId("key");
    await this.db.insert(schema.keys).values({
      id: keyId,
      keyAuthId: this.resources.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: this.resources.userWorkspace.id,
      createdAt: new Date(),
    });

    const addedPermissions = new Set<string>();
    for (const role of opts.roles) {
      const roleId = newId("role");

      await this.db.insert(schema.roles).values({
        id: roleId,
        name: role.name,
        createdAt: new Date(),
        workspaceId: this.resources.userWorkspace.id,
      });
      await this.db.insert(schema.keysRoles).values({
        keyId,
        roleId,
        workspaceId: this.resources.userWorkspace.id,
      });

      for (const permissionName of role.permissions ?? []) {
        if (addedPermissions.has(permissionName)) {
          continue;
        }
        addedPermissions.add(permissionName);

        const permissionId = newId("permission");
        await this.db.insert(schema.permissions).values({
          id: permissionId,
          name: permissionName,
          createdAt: new Date(),
          workspaceId: this.resources.userWorkspace.id,
        });
        await this.db.insert(schema.rolesPermissions).values({
          roleId,
          permissionId,
          workspaceId: this.resources.userWorkspace.id,
        });
      }
    }

    return {
      keyId,
      key,
    };
  }

  public createResources(): Resources {
    const unkeyWorkspace: Workspace = {
      id: newId("workspace"),
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
    };
    const userWorkspace: Workspace = {
      id: newId("workspace"),
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
    };

    const unkeyKeyAuth: KeyAuth = {
      id: newId("keyAuth"),
      workspaceId: unkeyWorkspace.id,
      createdAt: new Date(),
      deletedAt: null,
    };
    const userKeyAuth: KeyAuth = {
      id: newId("keyAuth"),
      workspaceId: userWorkspace.id,
      createdAt: new Date(),
      deletedAt: null,
    };

    const unkeyApi: Api = {
      id: newId("api"),
      name: "unkey",
      workspaceId: unkeyWorkspace.id,
      authType: "key",
      keyAuthId: unkeyKeyAuth.id,
      ipWhitelist: null,
      createdAt: new Date(),
      deletedAt: null,
    };
    const userApi: Api = {
      id: newId("api"),
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

  public async seed(): Promise<void> {
    if (this.seeded) {
      return;
    }

    await this.db.insert(schema.workspaces).values(this.resources.unkeyWorkspace);
    await this.db.insert(schema.keyAuth).values(this.resources.unkeyKeyAuth);
    await this.db.insert(schema.apis).values(this.resources.unkeyApi);

    await this.db.insert(schema.workspaces).values(this.resources.userWorkspace);
    await this.db.insert(schema.keyAuth).values(this.resources.userKeyAuth);
    await this.db.insert(schema.apis).values(this.resources.userApi);
    this.seeded = true;
  }
}
