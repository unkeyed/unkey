import type {
  Api,
  EncryptedKey,
  Identity,
  Key,
  KeyAuth,
  Ratelimit,
  RatelimitNamespace,
  RatelimitOverride,
} from "@unkey/db";

export type KeyHash = string;
export type CacheNamespaces = {
  keyById: {
    key: Key & { encrypted: EncryptedKey | null };
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
    key: Key & { encrypted: EncryptedKey | null };
    api: Api;
    permissions: string[];
    roles: string[];
    ratelimits: { [name: string]: Ratelimit };
    identity: Identity | null;
  } | null;
  apiById: (Api & { keyAuth: KeyAuth | null }) | null;
  keysByOwnerId: {
    key: Key & { encrypted: EncryptedKey | null };
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
  keysByApiId: {
    keys: Array<
      Key & {
        encrypted: EncryptedKey | null;
        permissions: string[];
        roles: string[];
      }
    >;
    total: number;
  };
};

export type CacheNamespace = keyof CacheNamespaces;
