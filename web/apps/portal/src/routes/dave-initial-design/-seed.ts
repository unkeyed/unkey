export type Key = {
  id: string;
  name: string | null;
  start: string;
  createdAt: number;
  expires: number | null;
  enabled: boolean;
  externalId: string;
  usage: number[];
  errors?: number[];
};

export type Branding = {
  logoUrl: string | null;
  backgroundColor: string;
  buttonColor: string;
  appName: string;
};

const DAY = 1000 * 60 * 60 * 24;
const now = Date.UTC(2026, 3, 23);

/**
 * 30 hourly buckets, oldest → newest. Distinct shapes per key so the sparkline
 * reads differently row-to-row during design iteration.
 */
const usagePatterns = {
  productionSteady: [
    420, 465, 380, 402, 445, 510, 488, 462, 405, 430, 455, 480, 512, 528, 504, 492, 475, 460, 445,
    430, 418, 405, 395, 380, 410, 442, 470, 498, 515, 532,
  ],
  productionErrors: [
    12, 8, 45, 15, 22, 18, 10, 14, 8, 12, 9, 7, 15, 20, 12, 10, 8, 14, 11, 9, 13, 16, 10, 8, 11, 14,
    9, 12, 8, 15,
  ],
  stagingModerate: [
    85, 92, 78, 0, 0, 110, 145, 132, 120, 0, 88, 105, 118, 140, 125, 0, 0, 94, 112, 128, 145, 138,
    120, 102, 0, 88, 115, 132, 148, 155,
  ],
  stagingErrors: [
    5, 2, 8, 0, 0, 4, 12, 3, 6, 0, 22, 18, 4, 2, 6, 0, 0, 3, 8, 12, 4, 6, 2, 5, 0, 3, 8, 4, 6, 2,
  ],
  devSpiky: [
    0, 0, 0, 0, 240, 180, 0, 0, 0, 0, 0, 0, 0, 320, 85, 0, 0, 0, 0, 0, 0, 0, 0, 410, 0, 0, 0, 0, 0,
    220,
  ],
  unnamedLow: [
    32, 28, 35, 40, 38, 30, 28, 33, 36, 42, 40, 35, 32, 28, 30, 38, 42, 45, 40, 36, 34, 32, 30, 35,
    38, 42, 45, 40, 38, 36,
  ],
  expiredDead: [
    15, 0, 0, 0, 8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
  ],
  disabledAllZero: [
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
  ],
  scratchRecent: [
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 12, 28, 42, 65, 88, 120, 145,
    180,
  ],
  edgeFlat: [
    185, 192, 178, 180, 195, 188, 190, 182, 186, 192, 188, 190, 185, 180, 192, 198, 190, 185, 188,
    192, 186, 180, 195, 200, 188, 192, 186, 190, 195, 188,
  ],
};

export const seedKeys: Key[] = [
  {
    id: "key_3ZsMzZ1K4eFp",
    name: "Production",
    start: "uk_live_4fA2",
    createdAt: now - 120 * DAY,
    expires: null,
    enabled: true,
    externalId: "user_aKk2mDp9",
    usage: usagePatterns.productionSteady,
    errors: usagePatterns.productionErrors,
  },
  {
    id: "key_7xQnW8LbV2dK",
    name: "Staging",
    start: "uk_test_9Bq1",
    createdAt: now - 60 * DAY,
    expires: now + 90 * DAY,
    enabled: true,
    externalId: "user_aKk2mDp9",
    usage: usagePatterns.stagingModerate,
    errors: usagePatterns.stagingErrors,
  },
  {
    id: "key_2dFmX9yTnP7s",
    name: "Local dev",
    start: "uk_test_11XY",
    createdAt: now - 14 * DAY,
    expires: now + 3 * DAY,
    enabled: true,
    externalId: "user_aKk2mDp9",
    usage: usagePatterns.devSpiky,
  },
  {
    id: "key_5hY8bPqZ3mLk",
    name: null,
    start: "uk_live_3c00",
    createdAt: now - 7 * DAY,
    expires: null,
    enabled: true,
    externalId: "user_aKk2mDp9",
    usage: usagePatterns.unnamedLow,
  },
  {
    id: "key_9cVbNmKqX2pY",
    name: "Old CI runner — retired January",
    start: "uk_test_RR77",
    createdAt: now - 400 * DAY,
    expires: now - 30 * DAY,
    enabled: true,
    externalId: "user_aKk2mDp9",
    usage: usagePatterns.expiredDead,
  },
  {
    id: "key_4tYpG2nKhXcM",
    name: "Paused integration",
    start: "uk_live_P42k",
    createdAt: now - 45 * DAY,
    expires: null,
    enabled: false,
    externalId: "user_aKk2mDp9",
    usage: usagePatterns.disabledAllZero,
  },
  {
    id: "key_8rMwKxZ3qPvN",
    name: "Scratch",
    start: "uk_test_x9k1",
    createdAt: now - 1 * DAY,
    expires: now + 30 * DAY,
    enabled: true,
    externalId: "user_aKk2mDp9",
    usage: usagePatterns.scratchRecent,
  },
  {
    id: "key_6qXcVbNmKwH2",
    name: "Edge worker",
    start: "uk_live_eDG8",
    createdAt: now - 220 * DAY,
    expires: null,
    enabled: true,
    externalId: "user_aKk2mDp9",
    usage: usagePatterns.edgeFlat,
  },
];

export const seedBranding: Branding = {
  logoUrl: null,
  backgroundColor: "#000000",
  buttonColor: "#000000",
  appName: "Acme Inc",
};
