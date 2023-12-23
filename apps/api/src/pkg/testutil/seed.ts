import { type Api, type KeyAuth, type Workspace, schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import { type Database, createConnection } from "../db";

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

export async function seed(env: {
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
