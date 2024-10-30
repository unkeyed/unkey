export type MessageBody = {
  migrationId: string;
  workspaceId: string;
  keyAuthId: string;
  rootKeyId: string;
  prefix?: string;

  name?: string;
  hash: string;
  start?: string;
  ownerId?: string;
  meta?: Record<string, unknown>;
  roles?: string[];
  permissions?: string[];
  expires?: number;
  remaining?: number;
  refill?: { interval: "daily" | "monthly"; amount: number; refillDay?: number };
  ratelimit?: { async: boolean; limit: number; duration: number };
  enabled: boolean;
  environment?: string;
  encrypted?: {
    encrypted: string;
    keyId: string;
  };

  auditLogContext: {
    location: string;
    userAgent: string;
  };
};
