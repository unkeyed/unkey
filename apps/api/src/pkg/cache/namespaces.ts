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

type CachedIdentity = Pick<Identity, "id" | "externalId" | "meta">;

export type CacheNamespaces = {
  keyById: {
    key: Key & { encrypted: EncryptedKey | null };
    api: Api;
    permissions: string[];
    roles: string[];
    identity: CachedIdentity | null;
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
    ratelimits: { [name: string]: Pick<Ratelimit, "name" | "limit" | "duration"> };
    identity: CachedIdentity | null;
  } | null;
  apiById: (Api & { keyAuth: KeyAuth | null }) | null;
  keysByOwnerId: {
    key: Key & { encrypted: EncryptedKey | null };
    api: Api;
  }[];
  verificationsByKeyId: {
    time: number;
    count: number;
    outcome: string;
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
        identity: CachedIdentity | null;
      }
    >;
    total: number;
  };
  identityByExternalId: Identity | null;
  identityById: Identity | null;
  // uses a compound key of [workspaceId, name]
  auditLogBucketByWorkspaceIdAndName: {
    id: string;
  };
  workspaceIdByRootKeyHash: string | null;
};

export type CacheNamespace = keyof CacheNamespaces;
