type Plan = "free" | "pro" | "enterprise";

export type Quotas = {
  maxActiveKeys: number | null;
  maxVerifications: number | null;
};

export const QUOTA: Record<Plan, Quotas> = {
  free: {
    maxActiveKeys: 100,
    maxVerifications: 2500,
  },
  pro: {
    maxActiveKeys: 100_000,
    maxVerifications: 100_000_000,
  },
  enterprise: {
    maxActiveKeys: null,
    maxVerifications: null,
  },
};
