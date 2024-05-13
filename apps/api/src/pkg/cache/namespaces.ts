import type { Api, Key, RatelimitNamespace, RatelimitOverride } from "@unkey/db";

export type KeyHash = string;
export type CacheNamespaces = {
  keyById: {
    key: Key;
    api: Api;
    permissions: string[];
    roles: string[];
  } | null;
  keyByHash: {
    workspace: {
      id: string;
      enabled: boolean;
    };
    forWorkspace: {
      id: string;
      enabled: boolean;
    } | null;
    key: Key;
    api: Api;
    permissions: string[];
    roles: string[];
  } | null;
  apiById: Api | null;
  keysByOwnerId: {
    key: Key;
    api: Api;
  }[];
  verificationsByKeyId: {
    time: number;
    success: number;
    rateLimited: number;
    usageExceeded: number;
  }[];
  ratelimitByIdentifier: {
    namespace: Pick<RatelimitNamespace, "id" | "workspaceId">;
    override?: Pick<RatelimitOverride, "async" | "duration" | "limit" | "sharding">;
  } | null;
};

export type CacheNamespace = keyof CacheNamespaces;
