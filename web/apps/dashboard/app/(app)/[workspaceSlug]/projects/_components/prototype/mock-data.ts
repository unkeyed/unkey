export type Scenario = "new" | "migrated" | "active" | "live";
export type MockScenario = Exclude<Scenario, "live">;
export type Timeframe = "24h" | "7d";

export type KeyspaceStat = {
  id: string;
  name: string;
  projectName: string;
  keyCount: number;
  requests: Record<Timeframe, number>;
  validPct: number;
  spark: Record<Timeframe, number[]>;
};

export type RatelimitStat = {
  id: string;
  name: string;
  projectName: string;
  checks: Record<Timeframe, number>;
  blockedPct: number;
  spark: Record<Timeframe, number[]>;
};

export type AppMock = { id: string; source: "github" | "code" };

export type ProjectMock = {
  id: string;
  name: string;
  subtitle: string;
  apps: AppMock[];
  appCount: number;
};

export type UsageStat = {
  billableTotal: number;
  quota: number;
  verifications: number;
  ratelimits: number;
  daysLeft: number;
  // Compute spend against included credits, in whole dollars.
  hasComputePlan: boolean;
  computeSpend: number;
  computeCredits: number;
};

export type OverviewData = {
  projects: ProjectMock[];
  keyspaces: KeyspaceStat[];
  ratelimits: RatelimitStat[];
  usage: UsageStat;
};

const migrated: OverviewData = {
  projects: [
    { id: "p_1", name: "acme-corp", subtitle: "Default project", apps: [], appCount: 0 },
  ],
  keyspaces: [
    {
      id: "ks_1",
      name: "production",
      projectName: "acme-corp",
      keyCount: 1284,
      requests: { "24h": 612_000, "7d": 4_190_000 },
      validPct: 99.4,
      spark: {
        "24h": [22, 24, 20, 26, 23, 30, 27, 33, 29, 35, 31, 38],
        "7d": [180, 210, 240, 205, 260, 300, 280],
      },
    },
    {
      id: "ks_2",
      name: "partner-api",
      projectName: "acme-corp",
      keyCount: 96,
      requests: { "24h": 41_000, "7d": 280_000 },
      validPct: 98.1,
      spark: {
        "24h": [20, 22, 16, 18, 12, 15, 9, 13, 8, 14, 10, 12],
        "7d": [30, 44, 38, 50, 42, 36, 48],
      },
    },
    {
      id: "ks_3",
      name: "staging",
      projectName: "acme-corp",
      keyCount: 312,
      requests: { "24h": 22_000, "7d": 154_000 },
      validPct: 97.0,
      spark: {
        "24h": [14, 18, 12, 20, 15, 22, 17, 24, 19, 16, 21, 18],
        "7d": [20, 26, 22, 30, 24, 28, 25],
      },
    },
  ],
  ratelimits: [
    {
      id: "rl_1",
      name: "auth.login",
      projectName: "acme-corp",
      checks: { "24h": 128_000, "7d": 910_000 },
      blockedPct: 3.2,
      spark: {
        "24h": [40, 44, 38, 50, 46, 54, 48, 60, 52, 58, 50, 62],
        "7d": [110, 130, 120, 150, 135, 128, 145],
      },
    },
    {
      id: "rl_2",
      name: "sms.send",
      projectName: "acme-corp",
      checks: { "24h": 40_000, "7d": 300_000 },
      blockedPct: 0.9,
      spark: {
        "24h": [12, 14, 10, 16, 13, 18, 15, 20, 14, 17, 12, 15],
        "7d": [38, 44, 40, 48, 42, 46, 41],
      },
    },
  ],
  usage: {
    billableTotal: 2_410_000,
    quota: 10_000_000,
    verifications: 1_900_000,
    ratelimits: 510_000,
    daysLeft: 18,
    hasComputePlan: false,
    computeSpend: 6,
    computeCredits: 20,
  },
};

const active: OverviewData = {
  projects: [
    {
      id: "p_a1",
      name: "acme-corp",
      subtitle: "Default project",
      apps: [{ id: "a1", source: "github" }, { id: "a2", source: "code" }],
      appCount: 3,
    },
    {
      id: "p_a2",
      name: "checkout-service",
      subtitle: "Deployed 2h ago · main",
      apps: [{ id: "a3", source: "github" }, { id: "a4", source: "code" }],
      appCount: 2,
    },
    {
      id: "p_a3",
      name: "docs-site",
      subtitle: "Deployed 1d ago · main",
      apps: [{ id: "a5", source: "github" }],
      appCount: 1,
    },
  ],
  keyspaces: [
    {
      id: "ks_a1",
      name: "production",
      projectName: "acme-corp",
      keyCount: 2140,
      requests: { "24h": 842_000, "7d": 5_910_000 },
      validPct: 99.6,
      spark: {
        "24h": [30, 34, 31, 40, 36, 46, 42, 52, 48, 44, 56, 60],
        "7d": [260, 300, 280, 340, 310, 360, 350],
      },
    },
    {
      id: "ks_a2",
      name: "checkout-keys",
      projectName: "checkout-service",
      keyCount: 220,
      requests: { "24h": 118_000, "7d": 812_000 },
      validPct: 99.1,
      spark: {
        "24h": [24, 28, 22, 30, 26, 34, 30, 38, 32, 36, 30, 40],
        "7d": [90, 110, 100, 130, 115, 125, 120],
      },
    },
    {
      id: "ks_a3",
      name: "partner-api",
      projectName: "acme-corp",
      keyCount: 96,
      requests: { "24h": 41_000, "7d": 280_000 },
      validPct: 98.1,
      spark: {
        "24h": [20, 22, 16, 18, 12, 15, 9, 13, 8, 14, 10, 12],
        "7d": [30, 44, 38, 50, 42, 36, 48],
      },
    },
  ],
  ratelimits: [
    {
      id: "rl_a1",
      name: "auth.login",
      projectName: "acme-corp",
      checks: { "24h": 128_000, "7d": 910_000 },
      blockedPct: 3.2,
      spark: {
        "24h": [40, 44, 38, 50, 46, 54, 48, 60, 52, 58, 50, 62],
        "7d": [110, 130, 120, 150, 135, 128, 145],
      },
    },
    {
      id: "rl_a2",
      name: "checkout.rate",
      projectName: "checkout-service",
      checks: { "24h": 64_000, "7d": 452_000 },
      blockedPct: 1.4,
      spark: {
        "24h": [18, 22, 16, 24, 20, 26, 22, 30, 24, 28, 22, 32],
        "7d": [58, 66, 60, 72, 64, 70, 62],
      },
    },
  ],
  usage: {
    billableTotal: 6_720_000,
    quota: 10_000_000,
    verifications: 5_800_000,
    ratelimits: 920_000,
    daysLeft: 18,
    hasComputePlan: true,
    computeSpend: 12.4,
    computeCredits: 20,
  },
};

const empty: OverviewData = {
  projects: [],
  keyspaces: [],
  ratelimits: [],
  usage: {
    billableTotal: 0,
    quota: 10_000_000,
    verifications: 0,
    ratelimits: 0,
    daysLeft: 30,
    hasComputePlan: false,
    computeSpend: 0,
    computeCredits: 20,
  },
};

export const MOCK: Record<MockScenario, OverviewData> = { new: empty, migrated, active };

export const SCENARIO_LABELS: Record<Scenario, string> = {
  new: "New",
  migrated: "Migrated",
  active: "Active",
  live: "Live",
};
