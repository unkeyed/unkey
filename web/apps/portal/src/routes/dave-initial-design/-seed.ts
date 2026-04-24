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

const patternKeys = Object.keys(usagePatterns) as (keyof typeof usagePatterns)[];

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
  {
    id: "key_aB3nM7kPqXcR",
    name: "Mobile iOS",
    start: "uk_live_mIos",
    createdAt: now - 85 * DAY,
    expires: null,
    enabled: true,
    externalId: "user_7xYp3kLq",
    usage: usagePatterns.productionSteady,
  },
  {
    id: "key_cD4pN8jRqYdT",
    name: "Mobile Android",
    start: "uk_live_mAnd",
    createdAt: now - 82 * DAY,
    expires: null,
    enabled: true,
    externalId: "user_7xYp3kLq",
    usage: usagePatterns.edgeFlat,
  },
  {
    id: "key_eF5qO9kSrZeU",
    name: "Backend worker",
    start: "uk_live_bwkr",
    createdAt: now - 150 * DAY,
    expires: null,
    enabled: true,
    externalId: "svc_prod_abc",
    usage: usagePatterns.productionSteady,
    errors: usagePatterns.productionErrors,
  },
  {
    id: "key_gH6rP0lTsAfV",
    name: "Admin dashboard",
    start: "uk_live_admn",
    createdAt: now - 200 * DAY,
    expires: null,
    enabled: true,
    externalId: "svc_prod_abc",
    usage: usagePatterns.stagingModerate,
  },
  {
    id: "key_iJ7sQ1mUtBgW",
    name: "Webhook processor",
    start: "uk_live_whok",
    createdAt: now - 95 * DAY,
    expires: null,
    enabled: true,
    externalId: "svc_prod_abc",
    usage: usagePatterns.edgeFlat,
  },
  {
    id: "key_kL8tR2nVuChX",
    name: "Analytics pipeline",
    start: "uk_live_anlx",
    createdAt: now - 300 * DAY,
    expires: null,
    enabled: true,
    externalId: "svc_prod_abc",
    usage: usagePatterns.productionSteady,
  },
  {
    id: "key_mN9uS3oWvDiY",
    name: "Import batch",
    start: "uk_test_imbt",
    createdAt: now - 50 * DAY,
    expires: now - 5 * DAY,
    enabled: true,
    externalId: "svc_staging_def",
    usage: usagePatterns.expiredDead,
  },
  {
    id: "key_oP0vT4pXwEjZ",
    name: "Export batch",
    start: "uk_test_exbt",
    createdAt: now - 40 * DAY,
    expires: now + 60 * DAY,
    enabled: true,
    externalId: "svc_staging_def",
    usage: usagePatterns.stagingModerate,
    errors: usagePatterns.stagingErrors,
  },
  {
    id: "key_qR1wU5qYxFkA",
    name: "Partner integration — Stripe",
    start: "uk_live_strp",
    createdAt: now - 180 * DAY,
    expires: null,
    enabled: true,
    externalId: "svc_prod_abc",
    usage: usagePatterns.productionSteady,
  },
  {
    id: "key_sT2xV6rZyGlB",
    name: "Partner integration — SendGrid",
    start: "uk_live_sndg",
    createdAt: now - 170 * DAY,
    expires: null,
    enabled: false,
    externalId: "svc_prod_abc",
    usage: usagePatterns.disabledAllZero,
  },
  {
    id: "key_uV3yW7sAzHmC",
    name: "CI runner",
    start: "uk_test_cirn",
    createdAt: now - 30 * DAY,
    expires: now + 120 * DAY,
    enabled: true,
    externalId: "svc_staging_def",
    usage: usagePatterns.stagingModerate,
  },
  {
    id: "key_wX4zX8tBaInD",
    name: "QA env",
    start: "uk_test_qaen",
    createdAt: now - 25 * DAY,
    expires: now + 90 * DAY,
    enabled: true,
    externalId: "svc_staging_def",
    usage: usagePatterns.devSpiky,
  },
  {
    id: "key_yZ5aY9uCbJoE",
    name: "Feature flag service",
    start: "uk_live_ffsv",
    createdAt: now - 110 * DAY,
    expires: null,
    enabled: true,
    externalId: "svc_prod_abc",
    usage: usagePatterns.edgeFlat,
  },
  {
    id: "key_bA6cZ0vDcKpF",
    name: "Billing cron",
    start: "uk_live_blcr",
    createdAt: now - 240 * DAY,
    expires: null,
    enabled: true,
    externalId: "svc_prod_abc",
    usage: usagePatterns.unnamedLow,
  },
  {
    id: "key_dB7eA1wEdLqG",
    name: "Email worker",
    start: "uk_live_emwk",
    createdAt: now - 95 * DAY,
    expires: null,
    enabled: true,
    externalId: "svc_prod_abc",
    usage: usagePatterns.productionSteady,
    errors: usagePatterns.stagingErrors,
  },
  {
    id: "key_fC8gB2xFeMrH",
    name: "Notification service",
    start: "uk_live_ntfy",
    createdAt: now - 75 * DAY,
    expires: null,
    enabled: true,
    externalId: "svc_prod_abc",
    usage: usagePatterns.edgeFlat,
  },
  {
    id: "key_hD9iC3yGfNsI",
    name: "Search indexer",
    start: "uk_live_srix",
    createdAt: now - 160 * DAY,
    expires: null,
    enabled: false,
    externalId: "svc_prod_abc",
    usage: usagePatterns.disabledAllZero,
  },
  {
    id: "key_jE0kD4zHgOtJ",
    name: "Image resizer",
    start: "uk_live_imrz",
    createdAt: now - 130 * DAY,
    expires: null,
    enabled: true,
    externalId: "svc_prod_abc",
    usage: usagePatterns.stagingModerate,
  },
  {
    id: "key_lF1mE5aIhPuK",
    name: "PDF generator",
    start: "uk_live_pdfg",
    createdAt: now - 210 * DAY,
    expires: null,
    enabled: true,
    externalId: "svc_prod_abc",
    usage: usagePatterns.unnamedLow,
  },
  {
    id: "key_nG2oF6bJiQvL",
    name: "Support tool",
    start: "uk_test_sprt",
    createdAt: now - 550 * DAY,
    expires: now - 200 * DAY,
    enabled: true,
    externalId: "user_7xYp3kLq",
    usage: usagePatterns.expiredDead,
  },
  {
    id: "key_pH3qG7cKjRwM",
    name: "Beta test key",
    start: "uk_test_beta",
    createdAt: now - 3 * DAY,
    expires: now + 7 * DAY,
    enabled: true,
    externalId: "user_7xYp3kLq",
    usage: usagePatterns.scratchRecent,
  },
  {
    id: "key_rI4sH8dLkSxN",
    name: "Partner API — Twilio",
    start: "uk_live_twlo",
    createdAt: now - 190 * DAY,
    expires: null,
    enabled: true,
    externalId: "svc_prod_abc",
    usage: usagePatterns.productionSteady,
  },
];

export const seedBranding: Branding = {
  logoUrl: null,
  backgroundColor: "#000000",
  buttonColor: "#000000",
  appName: "Acme Inc",
};

/**
 * Deterministic 25-row generator for the "worst case" preview state.
 * Same inputs always return the same rows so Prev/Next feels stable.
 * Filter/search is handled downstream by TanStack Table — this only controls
 * the pool of rows currently in memory.
 */
export function synthesizeKeys({
  page,
  count = 25,
}: {
  page: number;
  count?: number;
}): Key[] {
  const base = page * count;
  const rows: Key[] = [];
  for (let i = 0; i < count; i++) {
    const idx = base + i;
    const pattern = patternKeys[idx % patternKeys.length];
    const statusRoll = idx % 7;
    const enabled = statusRoll !== 5;
    const expired = statusRoll === 6;
    const externalRoll = idx % 4;
    const externalId = ["user_aKk2mDp9", "user_7xYp3kLq", "svc_prod_abc", "svc_staging_def"][
      externalRoll
    ] as string;
    rows.push({
      id: `key_${idx.toString(36).padStart(12, "0")}`,
      name: idx % 13 === 0 ? null : `Key #${idx.toLocaleString()}`,
      start: `uk_live_${idx.toString(36).padStart(4, "0").slice(0, 4)}`,
      createdAt: now - ((idx * 7) % 500) * DAY,
      expires: expired ? now - ((idx % 60) + 1) * DAY : idx % 3 === 0 ? now + 90 * DAY : null,
      enabled,
      externalId,
      usage: usagePatterns[pattern],
    });
  }
  return rows;
}
