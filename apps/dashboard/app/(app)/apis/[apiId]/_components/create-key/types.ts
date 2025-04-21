export type ProcessedFormData = {
  prefix: string | undefined;
  bytes: number;
  ownerId: string | undefined;
  name: string | undefined;
  environment: string | undefined;
  metaEnabled?: boolean;
  meta?: Record<string, unknown>;
  limitEnabled?: boolean;
  limit?: {
    remaining: number;
    refill?: {
      interval: "daily" | "monthly";
      amount: number;
      refillDay?: number;
    };
  };
  expireEnabled?: boolean;
  expires?: number;
  ratelimitEnabled?: boolean;
  ratelimit?: {
    async: boolean;
    duration: number;
    limit: number;
  };
};

export type SectionName = "general" | "ratelimit" | "credits" | "expiration" | "metadata";
export type SectionState = "valid" | "invalid" | "initial";
