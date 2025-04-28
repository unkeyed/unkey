export type ProcessedFormData = {
  prefix?: string;
  bytes: number;
  ownerId?: string;
  name?: string;
  environment?: string;
  meta?: Record<string, unknown>;
  remaining?: number;
  refill?: {
    amount: number;
    refillDay: number | null;
  };
  expires?: number;
  ratelimit?: {
    name: string;
    limit: number;
    refillInterval: number;
  }[];
  enabled: boolean;
};

export type SectionName = "general" | "ratelimit" | "credits" | "expiration" | "metadata";

export type SectionState = "valid" | "invalid" | "initial";
