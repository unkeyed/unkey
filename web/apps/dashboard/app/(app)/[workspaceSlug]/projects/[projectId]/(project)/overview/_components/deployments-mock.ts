import type { AppMock } from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/mock-data";
import {
  hashCode,
  mulberry32,
} from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/store";

export type DeploymentStatus = "ready" | "building" | "failed";

export type DeploymentMock = {
  id: string;
  appId: string;
  appName: string;
  appSource: AppMock["source"];
  branch: string;
  sha: string;
  status: DeploymentStatus;
  timeAgoMin: number;
  environment: "production" | "preview";
  message: string;
  actor: string;
};

// Pulled from unkeyed/unkey's real branch/commit history so the row can be
// stress-tested against actual lengths instead of tidy placeholder text.
const BRANCHES = [
  "feat/deploy-spend-cap",
  "fix/webhook-retry",
  "chronark/eng-3036-restate-worker-identity-verification",
  "MichaelUnkey/eng-3027-sentry-pii-scrubber-only-scrubs-url-fie",
  "eng-2850-stripe-billing-non-atomic-subscriptiondeleted-toctou-in",
  "dependabot/go_modules/github.com/oapi-codegen/oapi-codegen/v2-2.7.1",
];

const ACTORS = [
  "dave-hawkins",
  "chronark",
  "Flo4604",
  "mcstepp",
  "perkinsjr",
  "ogzhanolguncu",
  "MichaelUnkey",
  "dependabot[bot]",
  "mintlify[bot]",
];

const MESSAGE_POOLS = {
  worker: [
    "fix: kill cronjobs instead of pause and allow ratelimit cleanup to deal with 100k+ rows (#6844)",
    "feat(ctrl): rework build queue for per-env concurrency and cancellation (#5709)",
    "feat(ctrl): auto issue ssl cert on new region (#6708)",
    "fix(ctrl): debounce spend-cap auto-resume with two under-budget ticks (#6834)",
    "feat(ctrl): prune orphaned deployment resources",
    "refactor: make heartbeat client provider agnostic (#6821)",
  ],
  api: [
    "fix: oauth callback open redirect via unvalidated redirectUrlComplete in OAuth state (#6726)",
    "feat(api): expose rate limit analytics SQL (#6792)",
    "refactor(api,ctrl): move deploymentStatus and deploymentDesiredState from mysql pkg (#6793)",
    "feat(api): add environments list endpoint (#6534)",
    "fix(analytics): bound SQL result resources (#6798)",
    "test(api): stabilize key credit cache invalidation (#6833)",
  ],
  docs: [
    "docs: add ephemeral storage references across related pages (#5638)",
    "docs: update deployment beta instructions (#6822)",
    "docs: add changelog for July 10, 2026",
    "docs: add frontline clickhouse grants for request rollup tables (#6527)",
    "chore: update lockfile",
  ],
  web: [
    "feat(dashboard): estimated Compute bill and always-on usage (#6823)",
    "fix(dashboard): harden Stripe webhook handlers against reorder and partial failure (#6786)",
    "refactor(dashboard): Migrate ratelimits and identities to PageHeader layout (#6719)",
    "chore(dashboard): dev Stripe-seed button + build @unkey/api before dev (#6757)",
    "fix(dashboard): refresh workspace query on budget save so the paused banner updates",
    "feat(dashboard): drive runtime setting sliders from the workspace quota (#6707)",
  ],
  generic: [
    "chore: bump restate sdk for payload checks (#5953)",
    "fix: slack mrkdwn injection via attacker controlled customer name/email (#6768)",
    "both of these tests would accidentally use the same VO key and therefore cancel each other out sometimes. (#6835)",
    "refactor: simplify retry backoff",
    "chore: build and release individual images for each service (#6323)",
  ],
} as const;

// Buckets by keyword in the app name rather than a real "kind" field, since
// AppMock only carries source + name — good enough to vary commit flavor.
function messagePoolFor(appName: string): readonly string[] {
  const name = appName.toLowerCase();
  if (
    name.includes("worker") ||
    name.includes("webhook") ||
    name.includes("queue") ||
    name.includes("cron")
  ) {
    return MESSAGE_POOLS.worker;
  }
  if (name.includes("api")) {
    return MESSAGE_POOLS.api;
  }
  if (name.includes("doc")) {
    return MESSAGE_POOLS.docs;
  }
  if (name.includes("web") || name.includes("app") || name.includes("dashboard")) {
    return MESSAGE_POOLS.web;
  }
  return MESSAGE_POOLS.generic;
}

function rollBranch(rand: () => number): string {
  return rand() < 0.7 ? "main" : BRANCHES[Math.floor(rand() * BRANCHES.length)];
}

function rollStatus(rand: () => number): DeploymentStatus {
  const statusRoll = rand();
  return statusRoll < 0.08 ? "failed" : statusRoll < 0.22 ? "building" : "ready";
}

function rollSha(rand: () => number): string {
  return Array.from({ length: 7 }, () => Math.floor(rand() * 16).toString(16)).join("");
}

function pickMessage(rand: () => number, pool: readonly string[]): string {
  return pool[Math.floor(rand() * pool.length)];
}

function pickActor(rand: () => number): string {
  return ACTORS[Math.floor(rand() * ACTORS.length)];
}

// One deterministic "latest deployment" per app, seeded from the app id so it's
// stable across reloads without needing its own localStorage-backed store.
export function deploymentsForApps(apps: AppMock[]): DeploymentMock[] {
  return apps.map((app, i) => {
    const rand = mulberry32(hashCode(app.id));
    const branch = rollBranch(rand);
    const status = rollStatus(rand);
    const sha = rollSha(rand);
    const timeAgoMin =
      status === "building" ? 1 + Math.floor(rand() * 20) : (i + 1) * 60 + Math.floor(rand() * 300);
    const message = pickMessage(rand, messagePoolFor(app.name));
    const actor = pickActor(rand);
    return {
      id: `dep_${app.id}`,
      appId: app.id,
      appName: app.name,
      appSource: app.source,
      branch,
      sha,
      status,
      timeAgoMin,
      environment: branch === "main" ? "production" : "preview",
      message,
      actor,
    };
  });
}

// 5-8 past deployments per app, newest first. The rand stream is seeded purely
// from the app id (same as deploymentsForApps), so the first `branch`/`status`/
// `sha` draws land identically and history[0] always agrees with the single
// "latest deployment" deploymentsForApps returns for the same app.
export function deploymentHistoryForApp(app: AppMock, count?: number): DeploymentMock[] {
  const rand = mulberry32(hashCode(app.id));
  const pool = messagePoolFor(app.name);

  const branch = rollBranch(rand);
  const status = rollStatus(rand);
  const sha = rollSha(rand);
  const timeAgoMin =
    status === "building" ? 1 + Math.floor(rand() * 20) : 60 + Math.floor(rand() * 300);
  const message = pickMessage(rand, pool);
  const actor = pickActor(rand);
  const historyCount = count ?? 5 + Math.floor(rand() * 4);

  const first: DeploymentMock = {
    id: `dep_${app.id}`,
    appId: app.id,
    appName: app.name,
    appSource: app.source,
    branch,
    sha,
    status,
    timeAgoMin,
    environment: branch === "main" ? "production" : "preview",
    message,
    actor,
  };

  const items: DeploymentMock[] = [first];
  let cursor = first.timeAgoMin;

  for (let i = 1; i < historyCount; i++) {
    const olderBranch = rollBranch(rand);
    const olderStatus: DeploymentStatus = rand() < 0.12 ? "failed" : "ready";
    const olderSha = rollSha(rand);
    // Gaps widen further back: hours between recent deploys, days beyond that.
    const gapMin =
      i <= 2
        ? 45 + Math.floor(rand() * 180)
        : i <= 5
          ? 300 + Math.floor(rand() * 600)
          : 1440 + Math.floor(rand() * 2880);
    cursor += gapMin;

    items.push({
      id: `dep_${app.id}_${i}`,
      appId: app.id,
      appName: app.name,
      appSource: app.source,
      branch: olderBranch,
      sha: olderSha,
      status: olderStatus,
      timeAgoMin: cursor,
      environment: olderBranch === "main" ? "production" : "preview",
      message: pickMessage(rand, pool),
      actor: pickActor(rand),
    });
  }

  return items;
}

export function fmtTimeAgo(min: number): string {
  if (min < 60) {
    return `${min}m ago`;
  }
  const h = Math.round(min / 60);
  if (h < 24) {
    return `${h}h ago`;
  }
  return `${Math.round(h / 24)}d ago`;
}
