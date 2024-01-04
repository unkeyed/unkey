export type Quotas = {
  maxActiveKeys: number;
  maxVerifications: number;
};

export const QUOTA = {
  free: {
    maxActiveKeys: 100,
    maxVerifications: 2500,
  },
  pro: {
    maxActiveKeys: 100_000,
    maxVerifications: 100_000_000,
  },
} as const;
