import { type Api, type Database, type KeyAuth, type Workspace, schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import { createConnection } from "../db";

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
    slug: null,
    stripeCustomerId: null,
    stripeSubscriptionId: null,
    maxActiveKeys: 1000,
    usageActiveKeys: 0,
    maxVerifications: 1000,
    usageVerifications: 0,
    lastUsageUpdate: null,
    billingPeriodStart: null,
    billingPeriodEnd: null,
    trialEnds: null,
    subscriptions: null,
    planLockedUntil: null,
  };
  const userWorkspace: Workspace = {
    id: newId("workspace"),
    name: "user",
    tenantId: newId("test"),
    plan: "pro",
    slug: null,
    features: {},
    betaFeatures: {},
    stripeCustomerId: null,
    stripeSubscriptionId: null,
    maxActiveKeys: 1000,
    usageActiveKeys: 0,
    maxVerifications: 1000,
    usageVerifications: 0,
    lastUsageUpdate: null,
    billingPeriodStart: null,
    billingPeriodEnd: null,
    trialEnds: null,
    subscriptions: null,
    planLockedUntil: null,
  };

  const unkeyKeyAuth: KeyAuth = {
    id: newId("keyAuth"),
    workspaceId: unkeyWorkspace.id,
  };
  const userKeyAuth: KeyAuth = {
    id: newId("keyAuth"),
    workspaceId: userWorkspace.id,
  };

  const unkeyApi: Api = {
    id: newId("api"),
    name: "unkey",
    workspaceId: unkeyWorkspace.id,
    authType: "key",
    keyAuthId: unkeyKeyAuth.id,
    ipWhitelist: null,
  };
  const userApi: Api = {
    id: newId("api"),
    name: "user",
    workspaceId: userWorkspace.id,
    authType: "key",
    keyAuthId: userKeyAuth.id,
    ipWhitelist: null,
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
