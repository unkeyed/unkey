import type { Workspace } from "@unkey/db";

export type Plan = NonNullable<Workspace["plan"]>;

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
