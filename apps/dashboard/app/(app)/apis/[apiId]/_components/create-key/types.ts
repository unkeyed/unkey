export type ProcessedFormData = {
  prefix: string | null;
  bytes: number;
  ownerId: string | null;
  name: string | null;
  environment: string | null;
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
