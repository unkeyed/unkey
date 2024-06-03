import type {
  Api,
  EncryptedKey,
  Key,
  KeyAuth,
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
};

export type CacheNamespace = keyof CacheNamespaces;
