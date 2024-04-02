export type Quotas = {
  maxActiveKeys: number;
  maxVerifications: number;
  maxRatelimits: number;
};

export const QUOTA = {
  free: {
    maxActiveKeys: 100,
    maxVerifications: 2_500,
    maxRatelimits: 100_000,
  },
  pro: {
    maxActiveKeys: 100_000,
    maxVerifications: 100_000_000,
    maxRatelimits: 100_000_000,
  },
} satisfies Record<string, Quotas>;
