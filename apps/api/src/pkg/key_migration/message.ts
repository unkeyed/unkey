export type MessageBody = {
  migrationId: string;
  rootKeyId: string;
  prefix?: string;

  name?: string;
  plaintext?: string;
  hash?: string;
  start?: string;
  ownerId?: string;
  meta?: Record<string, unknown>;
  roles?: string[];
  permissions?: string[];
  expires?: number;
  remaining?: number;
  refill?: { interval: "daily" | "monthly"; amount: number };
  ratelimit?: { async: boolean; limit: number; duration: number };
  enabled: boolean;
  environment?: string;

  auditLogContext: {
    location: string;
    userAgent: string;
  };
};
