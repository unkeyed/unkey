import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import {
  type Api,
  type Database,
  type KeyAuth,
  type Workspace,
  createConnection,
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

export class Harness {
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

  public useRoutes(...registerFunctions: ((app: App) => any)[]): void {
    registerFunctions.forEach((fn) => fn(this.app));
  }

  async get<TRes>(req: Omit<StepRequest<never>, "method">): Promise<StepResponse<TRes>> {
    return await fetchRoute<never, TRes>(this.app, {
      method: "GET",
      ...req,
    });
  }
  async post<TReq, TRes>(req: Omit<StepRequest<TReq>, "method">): Promise<StepResponse<TRes>> {
    return await fetchRoute<TReq, TRes>(this.app, {
      method: "POST",
      ...req,
    });
  }
  async put<TReq, TRes>(req: Omit<StepRequest<TReq>, "method">): Promise<StepResponse<TRes>> {
    return await fetchRoute<TReq, TRes>(this.app, {
      method: "PUT",
      ...req,
    });
  }
  async delete<TRes>(req: Omit<StepRequest<never>, "method">): Promise<StepResponse<TRes>> {
    return await fetchRoute<never, TRes>(this.app, {
      method: "DELETE",
      ...req,
    });
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
      await this.resources.database.insert(schema.roles).values(
        roles.map((role) => ({
          id: newId("role"),
          workspaceId: this.resources.unkeyWorkspace.id,
          keyId,
          role,
        })),
      );
    }
    return {
      id: keyId,
      key: rootKey,
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
    state: null,
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
    state: null,
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
