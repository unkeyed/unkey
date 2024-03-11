import type { Api, Key, Ratelimit, RatelimitNamespace } from "@unkey/db";

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
    namespace: RatelimitNamespace;
    ratelimit?: Ratelimit;
  };
};
