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
  rootKey: string;
  userWorkspace: Workspace;
  userApi: Api;
  userKeyAuth: KeyAuth;
  database: Database;
};

export class Harness implements AsyncDisposable {
  public readonly db: Database;
  public readonly resources: Resources;
  public readonly app: App;

  private constructor(resources: Resources) {
    this.db = resources.database;
    this.resources = resources;
    this.app = newApp();
  }

  static async init(): Promise<Harness> {
    const env = unitTestEnv.parse(process.env);

    // @ts-ignore
    init({ env });
    const resources = await seed(env);

    return new Harness(resources);
  }

  async [Symbol.asyncDispose]() {
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
    for (const table of tables) {
      await this.db
        .delete(table)
        .where(
          or(
            eq(table.workspaceId, this.resources.userWorkspace.id),
            eq(table.workspaceId, this.resources.unkeyWorkspace.id),
          ),
        )
        .execute();
    }
    // this one is special, where the id is not prefixed
    await this.db
      .delete(schema.workspaces)
      .where(
        or(
          eq(schema.workspaces.id, this.resources.userWorkspace.id),
          eq(schema.workspaces.id, this.resources.unkeyWorkspace.id),
        ),
      )
      .execute();
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
  async createRootKey(roles?: string[]) {
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
    if (roles && roles.length > 0) {
      const permissions: Permission[] = roles.map((name) => ({
        id: newId("permission"),
        name,
        key: name,
        description: null,
        workspaceId: this.resources.unkeyWorkspace.id,
        createdAt: new Date(),
        updatedAt: null,
      }));

      await this.db.insert(schema.permissions).values(permissions);
      await this.db.insert(schema.keysPermissions).values(
        permissions.map((p) => ({
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
}

async function seed(env: {
  DATABASE_HOST: string;
  DATABASE_USERNAME: string;
  DATABASE_PASSWORD: string;
  DATABASE_MODE: "planetscale" | "mysql";
}): Promise<Resources> {
  const database = createConnection({
    host: env.DATABASE_HOST,
    username: env.DATABASE_USERNAME,
    password: env.DATABASE_PASSWORD,
  });

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

  await database.insert(schema.workspaces).values(unkeyWorkspace);
  await database.insert(schema.keyAuth).values(unkeyKeyAuth);
  await database.insert(schema.apis).values(unkeyApi);

  await database.insert(schema.workspaces).values(userWorkspace);
  await database.insert(schema.keyAuth).values(userKeyAuth);
  await database.insert(schema.apis).values(userApi);

  const rootKey = new KeyV1({ byteLength: 16, prefix: "unkey" }).toString();
  const start = rootKey.slice(0, 10);
  const keyId = newId("key");
  const hash = await sha256(rootKey);

  await database.insert(schema.keys).values({
    id: keyId,
    keyAuthId: unkeyKeyAuth.id,
    hash,
    start,
    workspaceId: unkeyWorkspace.id,
    forWorkspaceId: userWorkspace.id,
    createdAt: new Date(),
  });

  return {
    database,
    unkeyWorkspace,
    unkeyApi,
    unkeyKeyAuth,
    rootKey,
    userWorkspace,
    userApi,
    userKeyAuth,
  };
}
