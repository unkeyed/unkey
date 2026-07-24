import type {
  KeyspaceStat,
  RatelimitStat,
  Scenario,
} from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/mock-data";
import {
  hashCode,
  loadWorlds,
  type MockKey,
  mulberry32,
  SCENARIO_STORAGE_KEY,
  type World,
} from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/store";
import type { inferRouterOutputs } from "@trpc/server";
import { PASS, type PrototypeHandlers } from "./prototype-link";
import type { Router } from "./routers";

// Fake responses for the procedures behind the real keyspace/keys/ratelimit
// UIs, generated deterministically from the prototype store so every visitor
// sees the same data. Structural drift against the routers fails typecheck via
// the `satisfies Outputs[...]` clauses below.

type Outputs = inferRouterOutputs<Router>;

const WORKSPACE_ID = "ws_prototype";
const SCENARIOS: Scenario[] = ["new", "migrated", "active"];
const DAY_MS = 24 * 60 * 60 * 1000;
const HOUR_MS = 60 * 60 * 1000;

function activeScenario(): Scenario {
  try {
    const v = localStorage.getItem(SCENARIO_STORAGE_KEY);
    if (v === "new" || v === "migrated" || v === "active") {
      return v;
    }
  } catch {
    // ignore
  }
  return "migrated";
}

function activeWorld(): World {
  return loadWorlds()[activeScenario()];
}

function allWorlds(): World[] {
  const worlds = loadWorlds();
  return SCENARIOS.map((s) => worlds[s]);
}

// Keyspace stat ids double as the fake API ids; the keyAuth id is derived.
const toKeyAuthId = (apiId: string) => `kauth_${apiId}`;
const fromKeyAuthId = (keyAuthId: string) =>
  keyAuthId.startsWith("kauth_") ? keyAuthId.slice("kauth_".length) : keyAuthId;

function findKeyspace(apiId: string): { world: World; ks: KeyspaceStat; keys: MockKey[] } | null {
  for (const world of allWorlds()) {
    const ks = world.keyspaces.find((k) => k.id === apiId);
    if (ks) {
      return { world, ks, keys: world.keys[ks.id] ?? [] };
    }
  }
  return null;
}

function findNamespace(namespaceId: string): RatelimitStat | null {
  for (const world of allWorlds()) {
    const rl = world.ratelimits.find((r) => r.id === namespaceId);
    if (rl) {
      return rl;
    }
  }
  return null;
}

// Deterministic per-namespace identifier pool for ratelimit tables.
const IDENTIFIER_POOL = [
  "user_1042",
  "user_8821",
  "user_3319",
  "team_acme",
  "203.0.113.24",
  "198.51.100.9",
  "mobile_app",
  "api_7f3a",
  "partner_zapier",
  "batch_worker",
];

function namespaceIdentifiers(rl: RatelimitStat): string[] {
  const rand = mulberry32(hashCode(`ids:${rl.id}`));
  const count = 5 + Math.floor(rand() * 4);
  const pool = [...IDENTIFIER_POOL];
  for (let i = pool.length - 1; i > 0; i--) {
    const j = Math.floor(rand() * (i + 1));
    [pool[i], pool[j]] = [pool[j], pool[i]];
  }
  return pool.slice(0, count);
}

function findKey(keyId: string): { ks: KeyspaceStat; key: MockKey } | null {
  for (const world of allWorlds()) {
    for (const ks of world.keyspaces) {
      const key = (world.keys[ks.id] ?? []).find((k) => k.id === keyId);
      if (key) {
        return { ks, key };
      }
    }
  }
  return null;
}

const defaultPrefix = (ks: KeyspaceStat) =>
  ks.name.replace(/[^a-z0-9]/gi, "").slice(0, 4) || "key";

type FakeProject = World["projects"][number];

function findFakeProject(projectId: string | undefined): FakeProject | null {
  if (!projectId) {
    return null;
  }
  for (const world of allWorlds()) {
    const project = world.projects.find((p) => p.id === projectId);
    if (project) {
      return project;
    }
  }
  return null;
}

function findFakeApp(appId: string): { project: FakeProject; app: FakeProject["apps"][number] } | null {
  for (const world of allWorlds()) {
    for (const project of world.projects) {
      const app = project.apps.find((a) => a.id === appId);
      if (app) {
        return { project, app };
      }
    }
  }
  return null;
}

// Fake projects have no custom domains; short-circuit that list to empty
// instead of letting the backend 404 on unknown project ids.
const emptyForFakeProject = (rawInput: unknown) => {
  const input = rawInput as { projectId?: string };
  return findFakeProject(input.projectId) ? [] : PASS;
};

// ---------------------------------------------------------------------------
// Deploy-tree fabrication (apps, environments, deployments, domains) so the
// project Apps flow and app overview render for fake projects.
// ---------------------------------------------------------------------------

const REGION = { id: "reg_us_east_1", name: "us-east-1", platform: "aws" };
const COMMIT_TITLES = [
  "fix: retry webhook delivery on 5xx",
  "feat: add usage caps to checkout",
  "chore: bump dependencies",
  "feat: stream responses from the edge",
  "fix: handle empty cart on checkout",
  "refactor: split billing worker queue",
];

const projectSlug = (project: FakeProject) =>
  project.name.toLowerCase().replace(/[^a-z0-9-]/g, "-");

const currentDeploymentId = (appId: string) => `dep_${appId}_current`;
const productionEnvId = (appId: string) => `env_${appId}_production`;

function appCommit(appId: string) {
  const rand = mulberry32(hashCode(`commit:${appId}`));
  const sha = Array.from({ length: 40 }, () => Math.floor(rand() * 16).toString(16)).join("");
  return {
    title: COMMIT_TITLES[Math.floor(rand() * COMMIT_TITLES.length)],
    sha,
    // 1h–2d ago, stable per app within a session.
    timestamp: Date.now() - Math.floor((1 + rand() * 47) * HOUR_MS),
  };
}

const appDomain = (project: FakeProject, app: FakeProject["apps"][number]) =>
  `${app.name}-${projectSlug(project)}.unkey.app`;

function fakeAppRows(project: FakeProject): Outputs["deploy"]["app"]["list"] {
  return project.apps.map((app) => {
    const commit = appCommit(app.id);
    return {
      id: app.id,
      projectId: project.id,
      name: app.name,
      slug: app.name,
      defaultBranch: "main",
      currentDeploymentId: currentDeploymentId(app.id),
      isRolledBack: false,
      repositoryFullName: app.source === "github" ? `acme/${projectSlug(project)}` : null,
      latestDeploymentId: currentDeploymentId(app.id),
      commitTitle: commit.title,
      commitSha: commit.sha,
      forkRepositoryFullName: null,
      prNumber: null,
      branch: "main",
      author: app.source === "github" ? "dave" : null,
      authorAvatar: null,
      commitTimestamp: commit.timestamp,
      domain: appDomain(project, app),
    };
  });
}

function fakeEnvironmentRows(project: FakeProject): Outputs["deploy"]["environment"]["list"] {
  return project.apps.flatMap((app) => [
    { id: productionEnvId(app.id), projectId: project.id, slug: "production", appId: app.id },
    { id: `env_${app.id}_preview`, projectId: project.id, slug: "preview", appId: app.id },
  ]);
}

function fakeDeploymentRows(project: FakeProject): Outputs["deploy"]["deployment"]["list"] {
  return project.apps.flatMap((app) => {
    const commit = appCommit(app.id);
    const base = {
      projectId: project.id,
      appId: app.id,
      environmentId: productionEnvId(app.id),
      gitBranch: "main",
      gitCommitAuthorHandle: app.source === "github" ? "dave" : null,
      gitCommitAuthorAvatarUrl: "",
      prNumber: null,
      forkRepositoryFullName: null,
      image: app.source === "code" ? `registry.unkey.app/${app.name}:latest` : null,
      hasOpenApiSpec: false,
      desiredState: "running" as const,
      desiredInstanceCount: 1,
      desiredRegions: [{ region: REGION, flagCode: "us" as const }],
      cpuMillicores: 250,
      memoryMib: 512,
      storageMib: 1024,
      port: 8080,
      upstreamProtocol: "http1" as const,
      healthcheck: null,
      shutdownSignal: "SIGTERM" as const,
      trigger: app.source === "github" ? ("github" as const) : ("cli" as const),
      triggeredBy: "dave",
      triggerReason: null,
      updatedAt: null,
      lastExit: null,
    };
    const previous = appCommit(`${app.id}:prev`);
    return [
      {
        ...base,
        id: currentDeploymentId(app.id),
        gitCommitSha: commit.sha,
        gitCommitMessage: commit.title,
        gitCommitTimestamp: commit.timestamp,
        status: "ready" as const,
        instances: [
          {
            id: `inst_${app.id}_1`,
            region: REGION,
            flagCode: "us" as const,
            status: "running" as const,
          },
        ],
        createdAt: commit.timestamp,
        buildEndedAt: commit.timestamp + 95_000,
      },
      {
        ...base,
        id: `dep_${app.id}_prev`,
        gitCommitSha: previous.sha,
        gitCommitMessage: previous.title,
        gitCommitTimestamp: commit.timestamp - 26 * HOUR_MS,
        status: "superseded" as const,
        instances: [],
        createdAt: commit.timestamp - 26 * HOUR_MS,
        buildEndedAt: commit.timestamp - 26 * HOUR_MS + 80_000,
      },
    ];
  });
}

function fakeDomainRows(project: FakeProject): Outputs["deploy"]["domain"]["list"] {
  return project.apps.map((app) => ({
    id: `dom_${app.id}`,
    fullyQualifiedDomainName: appDomain(project, app),
    projectId: project.id,
    appId: app.id,
    deploymentId: currentDeploymentId(app.id),
    environmentId: productionEnvId(app.id),
    sticky: "live" as const,
    createdAt: appCommit(app.id).timestamp,
    updatedAt: null,
  }));
}

// ---------------------------------------------------------------------------
// Timeseries generation
// ---------------------------------------------------------------------------

type Granularity = Outputs["api"]["overview"]["timeseries"]["granularity"];

function pickGranularity(durationMs: number): { granularity: Granularity; stepMs: number } {
  if (durationMs <= HOUR_MS) {
    return { granularity: "perMinute", stepMs: 60 * 1000 };
  }
  if (durationMs <= 6 * HOUR_MS) {
    return { granularity: "per5Minutes", stepMs: 5 * 60 * 1000 };
  }
  if (durationMs <= 12 * HOUR_MS) {
    return { granularity: "per15Minutes", stepMs: 15 * 60 * 1000 };
  }
  if (durationMs <= DAY_MS) {
    return { granularity: "per30Minutes", stepMs: 30 * 60 * 1000 };
  }
  if (durationMs <= 3 * DAY_MS) {
    return { granularity: "per2Hours", stepMs: 2 * HOUR_MS };
  }
  if (durationMs <= 7 * DAY_MS) {
    return { granularity: "per4Hours", stepMs: 4 * HOUR_MS };
  }
  if (durationMs <= 31 * DAY_MS) {
    return { granularity: "perDay", stepMs: DAY_MS };
  }
  return { granularity: "perWeek", stepMs: 7 * DAY_MS };
}

// Normalizes the requested window: honors a relative `since` when set, and
// clamps degenerate inputs so we never fabricate thousands of buckets.
function resolveWindow(input: { startTime?: number; endTime?: number; since?: string }): {
  startTime: number;
  endTime: number;
} {
  const endTime = input.endTime && input.endTime > 0 ? input.endTime : Date.now();
  let startTime = input.startTime && input.startTime > 0 ? input.startTime : endTime - DAY_MS;
  const since = (input.since ?? "").trim();
  const match = since.match(/^(\d+)([mhdw])$/);
  if (match) {
    const n = Number(match[1]);
    const unit = { m: 60 * 1000, h: HOUR_MS, d: DAY_MS, w: 7 * DAY_MS }[match[2]] ?? HOUR_MS;
    startTime = endTime - n * unit;
  }
  if (startTime >= endTime || endTime - startTime > 90 * DAY_MS) {
    startTime = endTime - DAY_MS;
  }
  return { startTime, endTime };
}

// Relative intensity of a moment in the day, from the keyspace's 24h spark
// shape, so charts have a believable daily rhythm.
function curveAt(spark: number[], tMs: number): number {
  if (spark.length === 0) {
    return 1;
  }
  const pos = ((tMs % DAY_MS) / DAY_MS) * (spark.length - 1);
  const lo = Math.floor(pos);
  const hi = Math.min(spark.length - 1, lo + 1);
  const frac = pos - lo;
  const value = spark[lo] * (1 - frac) + spark[hi] * frac;
  const avg = spark.reduce((a, b) => a + b, 0) / spark.length;
  return avg > 0 ? value / avg : 1;
}

type VerificationPoint = NonNullable<
  Outputs["api"]["overview"]["timeseries"]["timeseries"]
>[number];

function verificationSeries(
  seedKey: string,
  spark: number[],
  total24h: number,
  validPct: number,
  window: { startTime?: number; endTime?: number; since?: string },
): { timeseries: VerificationPoint[]; granularity: Granularity } {
  const { startTime, endTime } = resolveWindow(window);
  const { granularity, stepMs } = pickGranularity(endTime - startTime);
  const perBucketBase = total24h * (stepMs / DAY_MS);
  const errorShare = Math.max(0, (100 - validPct) / 100);

  const points: VerificationPoint[] = [];
  for (let x = Math.floor(startTime / stepMs) * stepMs; x <= endTime; x += stepMs) {
    const rand = mulberry32(hashCode(`${seedKey}:${x}`));
    const total = Math.max(0, Math.round(perBucketBase * curveAt(spark, x) * (0.7 + rand() * 0.6)));
    const errors = Math.min(total, Math.round(total * errorShare * (0.5 + rand() * 1.2)));
    const valid = total - errors;
    const rateLimited = Math.round(errors * 0.55);
    const expired = Math.round(errors * 0.15);
    const disabled = Math.round(errors * 0.1);
    const insufficient = Math.round(errors * 0.08);
    const forbidden = Math.round(errors * 0.05);
    const usageExceeded = Math.max(
      0,
      errors - rateLimited - expired - disabled - insufficient - forbidden,
    );
    points.push({
      x,
      y: {
        total,
        valid,
        valid_count: valid,
        rate_limited_count: rateLimited,
        insufficient_permissions_count: insufficient,
        forbidden_count: forbidden,
        disabled_count: disabled,
        expired_count: expired,
        usage_exceeded_count: usageExceeded,
      },
    });
  }
  return { timeseries: points, granularity };
}

// ---------------------------------------------------------------------------
// Row fabrication
// ---------------------------------------------------------------------------

function toApiListItem(world: World) {
  return (ks: KeyspaceStat) => ({
    id: ks.id,
    name: ks.name,
    keyspaceId: toKeyAuthId(ks.id),
    keyCount: (world.keys[ks.id] ?? []).length,
  });
}

function toKeyListRow(key: MockKey): Outputs["api"]["keys"]["list"]["keys"][number] {
  return {
    id: key.id,
    name: key.name,
    owner_id: null,
    identity_id: null,
    enabled: key.enabled,
    expires: null,
    identity: null,
    updated_at_m: null,
    metadata: null,
    start: key.start,
    last_used_at: Date.now() - key.lastUsedMinAgo * 60 * 1000,
    key: {
      credits: { enabled: false, remaining: null, refillAmount: null, refillDay: null },
      ratelimits: { enabled: false, items: [] },
    },
  };
}

function toOverviewLogRow(
  ks: KeyspaceStat,
  key: MockKey,
): Outputs["api"]["keys"]["query"]["keysOverviewLogs"][number] {
  const rand = mulberry32(hashCode(`log:${key.id}`));
  const errors = Math.round(key.requests24h * ((100 - key.validPct) / 100));
  const valid = key.requests24h - errors;
  const rateLimited = Math.round(errors * 0.6);
  const expired = Math.max(0, errors - rateLimited);
  const outcomeCounts: Record<string, number> = { VALID: valid };
  if (rateLimited > 0) {
    outcomeCounts.RATE_LIMITED = rateLimited;
  }
  if (expired > 0) {
    outcomeCounts.EXPIRED = expired;
  }
  return {
    time: Date.now() - key.lastUsedMinAgo * 60 * 1000,
    key_id: key.id,
    request_id: `req_${Math.floor(rand() * 0xffffffff).toString(16)}`,
    valid_count: valid,
    error_count: errors,
    outcome_counts: outcomeCounts,
    tags: [],
    key_details: {
      id: key.id,
      key_auth_id: toKeyAuthId(ks.id),
      name: key.name,
      owner_id: null,
      identity_id: null,
      meta: null,
      enabled: key.enabled,
      remaining_requests: null,
      environment: null,
      workspace_id: WORKSPACE_ID,
      identity: null,
      roles: [],
      permissions: [],
    },
  };
}

function fakeProjects(): Outputs["deploy"]["project"]["list"] {
  const seen = new Set<string>();
  const out: Outputs["deploy"]["project"]["list"] = [];
  for (const world of allWorlds()) {
    for (const project of world.projects) {
      if (seen.has(project.id)) {
        continue;
      }
      seen.add(project.id);
      out.push({
        id: project.id,
        name: project.name,
        slug: projectSlug(project),
        appCount: project.appCount,
        apps: project.apps.map((app) => ({
          id: app.id,
          name: app.name,
          source: app.source,
          repository: app.source === "github" ? `acme/${projectSlug(project)}` : null,
        })),
        repositoryFullName: null,
        latestDeploymentId: null,
        currentDeploymentId: null,
        isRolledBack: false,
        commitTitle: null,
        commitSha: null,
        forkRepositoryFullName: null,
        prNumber: null,
        branch: "main",
        author: null,
        authorAvatar: null,
        commitTimestamp: null,
        domain: null,
      });
    }
  }
  return out;
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

const EMPTY_SERIES = { timeseries: [], granularity: "perHour" as Granularity };

export const prototypeHandlers: PrototypeHandlers = {
  replace: {
    "deploy.customDomain.list": emptyForFakeProject,

    "deploy.app.list": (rawInput) => {
      const project = findFakeProject((rawInput as { projectId?: string }).projectId);
      return project ? fakeAppRows(project) : PASS;
    },

    "deploy.environment.list": (rawInput) => {
      const project = findFakeProject((rawInput as { projectId?: string }).projectId);
      return project ? fakeEnvironmentRows(project) : PASS;
    },

    "deploy.deployment.list": (rawInput) => {
      const input = rawInput as { projectId?: string; appId?: string };
      const project = findFakeProject(input.projectId);
      if (!project) {
        return PASS;
      }
      const rows = fakeDeploymentRows(project);
      return input.appId ? rows.filter((d) => d.appId === input.appId) : rows;
    },

    "deploy.domain.list": (rawInput) => {
      const input = rawInput as { projectId?: string; appId?: string };
      const project = findFakeProject(input.projectId);
      if (!project) {
        return PASS;
      }
      const rows = fakeDomainRows(project);
      return input.appId ? rows.filter((d) => d.appId === input.appId) : rows;
    },

    "deploy.metrics.getAppRpsMetrics": (rawInput) => {
      const input = rawInput as { appId: string };
      const found = findFakeApp(input.appId);
      if (!found) {
        return PASS;
      }
      // Week view: 168 hourly buckets of average req/s, like a >1d-old app.
      const bucketMs = HOUR_MS;
      const end = Math.floor(Date.now() / bucketMs) * bucketMs;
      const baseRps = 4 + (Math.abs(hashCode(input.appId)) % 40);
      let totalRequests = 0;
      const rps: Array<{ time: number; requests: number; errors: number }> = [];
      for (let i = 168; i >= 1; i--) {
        const time = end - i * bucketMs;
        const rand = mulberry32(hashCode(`${input.appId}:rps:${time}`));
        const daily = 0.6 + 0.8 * Math.abs(Math.sin(((time % DAY_MS) / DAY_MS) * Math.PI));
        const requests = Math.round(baseRps * daily * (0.8 + rand() * 0.4) * 100) / 100;
        const errors = Math.round(requests * rand() * 0.02 * 100) / 100;
        totalRequests += Math.round(requests * (bucketMs / 1000));
        rps.push({ time, requests, errors });
      }
      return {
        range: "week",
        totalRequests,
        rps,
      } satisfies Outputs["deploy"]["metrics"]["getAppRpsMetrics"];
    },

    "api.overview.query": (rawInput) => {
      const input = rawInput as { limit?: number; cursor?: { id: string } };
      const world = activeWorld();
      const list = world.keyspaces.map(toApiListItem(world));
      // The dataset is small; serve everything on the first page.
      const apiList = input.cursor ? [] : list;
      return {
        apiList,
        hasMore: false,
        total: list.length,
      } satisfies Outputs["api"]["overview"]["query"];
    },

    "api.overview.search": (rawInput) => {
      const input = rawInput as { query: string };
      const q = input.query.trim().toLowerCase();
      const world = activeWorld();
      return world.keyspaces
        .filter((ks) => ks.name.toLowerCase().includes(q) || ks.id.toLowerCase().includes(q))
        .map(toApiListItem(world)) satisfies Outputs["api"]["overview"]["search"];
    },

    "api.overview.timeseries": (rawInput) => {
      const input = rawInput as {
        startTime: number;
        endTime: number;
        since: string;
        keyspaceId: string;
      };
      const found = findKeyspace(fromKeyAuthId(input.keyspaceId));
      if (!found) {
        return EMPTY_SERIES satisfies Outputs["api"]["overview"]["timeseries"];
      }
      return verificationSeries(
        found.ks.id,
        found.ks.spark["24h"],
        found.ks.requests["24h"],
        found.ks.validPct,
        input,
      ) satisfies Outputs["api"]["overview"]["timeseries"];
    },

    "api.queryApiKeyDetails": (rawInput) => {
      const input = rawInput as { apiId: string };
      const found = findKeyspace(input.apiId);
      if (!found) {
        throw notFoundError(`API ${input.apiId} not found`);
      }
      const { world, ks, keys } = found;
      return {
        currentApi: {
          id: ks.id,
          name: ks.name,
          workspaceId: WORKSPACE_ID,
          keyAuthId: toKeyAuthId(ks.id),
          keyspaceDefaults: null,
          deleteProtection: true,
          ipWhitelist: null,
        },
        workspaceApis: world.keyspaces.map((k) => ({ id: k.id, name: k.name })),
        keyAuth: {
          id: toKeyAuthId(ks.id),
          defaultPrefix: defaultPrefix(ks),
          defaultBytes: 16,
          sizeApprox: keys.length,
        },
        workspace: { id: WORKSPACE_ID },
      } satisfies Outputs["api"]["queryApiKeyDetails"];
    },

    "api.keys.list": (rawInput) => {
      const input = rawInput as {
        keyAuthId: string;
        names?: Array<{ value: string }> | null;
        keyIds?: Array<{ value: string }> | null;
      };
      const found = findKeyspace(fromKeyAuthId(input.keyAuthId));
      if (!found) {
        return { keys: [], totalCount: 0 } satisfies Outputs["api"]["keys"]["list"];
      }
      let keys = found.keys;
      if (input.names?.length) {
        const values = input.names.map((f) => f.value.toLowerCase());
        keys = keys.filter((k) => values.some((v) => k.name.toLowerCase().includes(v)));
      }
      if (input.keyIds?.length) {
        const values = input.keyIds.map((f) => f.value.toLowerCase());
        keys = keys.filter((k) => values.some((v) => k.id.toLowerCase().includes(v)));
      }
      return {
        keys: keys.map(toKeyListRow),
        totalCount: keys.length,
      } satisfies Outputs["api"]["keys"]["list"];
    },

    "api.keys.query": (rawInput) => {
      const input = rawInput as { apiId: string };
      const found = findKeyspace(input.apiId);
      if (!found) {
        return {
          keysOverviewLogs: [],
          hasMore: false,
          total: 0,
        } satisfies Outputs["api"]["keys"]["query"];
      }
      const rows = found.keys
        .map((key) => toOverviewLogRow(found.ks, key))
        .sort((a, b) => b.time - a.time);
      return {
        keysOverviewLogs: rows,
        hasMore: false,
        total: rows.length,
      } satisfies Outputs["api"]["keys"]["query"];
    },

    "api.keys.timeseries": (rawInput) => {
      const input = rawInput as {
        startTime: number;
        endTime: number;
        since: string;
        apiId: string;
      };
      const found = findKeyspace(input.apiId);
      if (!found) {
        return EMPTY_SERIES satisfies Outputs["api"]["keys"]["timeseries"];
      }
      return verificationSeries(
        found.ks.id,
        found.ks.spark["24h"],
        found.ks.requests["24h"],
        found.ks.validPct,
        input,
      ) satisfies Outputs["api"]["keys"]["timeseries"];
    },

    "api.keys.activeKeysTimeseries": (rawInput) => {
      const input = rawInput as {
        startTime: number;
        endTime: number;
        since: string;
        apiId: string;
      };
      const found = findKeyspace(input.apiId);
      if (!found) {
        return {
          timeseries: null,
          granularity: "perHour",
        } satisfies Outputs["api"]["keys"]["activeKeysTimeseries"];
      }
      const { startTime, endTime } = resolveWindow(input);
      const { granularity, stepMs } = pickGranularity(endTime - startTime);
      const activeKeys = found.keys.filter((k) => k.enabled).length;
      const points: Array<{ x: number; y: { keys: number } }> = [];
      for (let x = Math.floor(startTime / stepMs) * stepMs; x <= endTime; x += stepMs) {
        const rand = mulberry32(hashCode(`${found.ks.id}:active:${x}`));
        const share = 0.4 + 0.6 * Math.min(1.5, curveAt(found.ks.spark["24h"], x)) * rand();
        points.push({ x, y: { keys: Math.max(1, Math.round(activeKeys * share)) } });
      }
      return {
        timeseries: points,
        granularity,
      } satisfies Outputs["api"]["keys"]["activeKeysTimeseries"];
    },

    "api.keys.usageTimeseries": (rawInput) => {
      const input = rawInput as { startTime: number; endTime: number; keyId: string };
      const found = findKey(input.keyId);
      if (!found) {
        return { timeseries: [] } satisfies Outputs["api"]["keys"]["usageTimeseries"];
      }
      const { timeseries } = verificationSeries(
        found.key.id,
        found.key.spark,
        found.key.requests24h,
        found.key.validPct,
        input,
      );
      return { timeseries } satisfies Outputs["api"]["keys"]["usageTimeseries"];
    },

    "key.logs.timeseries": (rawInput) => {
      const input = rawInput as {
        startTime: number;
        endTime: number;
        since: string;
        keyId: string;
      };
      const found = findKey(input.keyId);
      if (!found) {
        return EMPTY_SERIES satisfies Outputs["key"]["logs"]["timeseries"];
      }
      return verificationSeries(
        found.key.id,
        found.key.spark,
        found.key.requests24h,
        found.key.validPct,
        input,
      ) satisfies Outputs["key"]["logs"]["timeseries"];
    },

    "key.logs.query": (rawInput) => {
      const input = rawInput as { keyId: string; limit?: number; page?: number };
      const found = findKey(input.keyId);
      if (!found) {
        return { logs: [], total: 0 } satisfies Outputs["key"]["logs"]["query"];
      }
      const limit = input.limit ?? 50;
      const page = input.page ?? 1;
      const totalRows = Math.min(200, Math.max(30, Math.round(found.key.requests24h / 500)));
      const start = (page - 1) * limit;
      const errorShare = (100 - found.key.validPct) / 100;
      const regions = ["us-east-1", "eu-west-1", "ap-southeast-1"] as const;
      const errorOutcomes = ["RATE_LIMITED", "EXPIRED", "DISABLED"] as const;
      const logs = [];
      for (let i = start; i < Math.min(start + limit, totalRows); i++) {
        const rand = mulberry32(hashCode(`${found.key.id}:kl:${i}`));
        const failed = rand() < errorShare;
        logs.push({
          request_id: `req_${Math.floor(rand() * 0xffffffff).toString(16)}${i}`,
          time: Date.now() - Math.floor(i * 3 * 60 * 1000 + rand() * 2 * 60 * 1000),
          region: regions[Math.floor(rand() * regions.length)],
          outcome: failed
            ? errorOutcomes[Math.floor(rand() * errorOutcomes.length)]
            : ("VALID" as const),
          tags: [],
        });
      }
      return { logs, total: totalRows } satisfies Outputs["key"]["logs"]["query"];
    },

    "key.fetchPermissions": (rawInput) => {
      const input = rawInput as { keyId: string; keyspaceId: string };
      const found = findKey(input.keyId);
      const ks = found?.ks;
      const createdAt = Date.now() - 30 * DAY_MS;
      return {
        keyId: input.keyId,
        keyAuth: {
          pk: 1,
          id: input.keyspaceId,
          workspaceId: WORKSPACE_ID,
          createdAtM: createdAt,
          updatedAtM: null,
          deletedAtM: null,
          storeEncryptedKeys: false,
          defaultPrefix: ks ? defaultPrefix(ks) : null,
          defaultBytes: 16,
          sizeApprox: ks ? (findKeyspace(ks.id)?.keys.length ?? 0) : 0,
          sizeLastUpdatedAt: createdAt,
        },
        roles: [],
        directPermissions: [],
        workspace: { roles: [], permissions: { roles: [] } },
        remainingCredit: null,
      } satisfies Outputs["key"]["fetchPermissions"];
    },

    "ratelimit.logs.queryRatelimitTimeseries": (rawInput) => {
      const input = rawInput as {
        startTime: number;
        endTime: number;
        since: string;
        namespaceId: string;
      };
      const rl = findNamespace(input.namespaceId);
      if (!rl) {
        return {
          timeseries: [],
          granularity: "perMinute",
        } satisfies Outputs["ratelimit"]["logs"]["queryRatelimitTimeseries"];
      }
      const { startTime, endTime } = resolveWindow(input);
      const { granularity, stepMs } = pickGranularity(endTime - startTime);
      const perBucketBase = rl.checks["24h"] * (stepMs / DAY_MS);
      const blockedShare = rl.blockedPct / 100;
      const timeseries = [];
      for (let x = Math.floor(startTime / stepMs) * stepMs; x <= endTime; x += stepMs) {
        const rand = mulberry32(hashCode(`${rl.id}:${x}`));
        const total = Math.max(
          0,
          Math.round(perBucketBase * curveAt(rl.spark["24h"], x) * (0.7 + rand() * 0.6)),
        );
        const blocked = Math.min(total, Math.round(total * blockedShare * (0.5 + rand() * 1.2)));
        const passed = total - blocked;
        timeseries.push({
          x,
          y: { passed, total, passed_tokens: passed, total_tokens: total },
        });
      }
      return {
        timeseries,
        granularity,
      } satisfies Outputs["ratelimit"]["logs"]["queryRatelimitTimeseries"];
    },

    "ratelimit.logs.queryRatelimitTimeseriesBatch": (rawInput) => {
      const input = rawInput as { namespaceIds: string[]; startTime: number; endTime: number };
      const { startTime, endTime } = resolveWindow(input);
      const { granularity, stepMs } = pickGranularity(endTime - startTime);
      const timeseriesByNamespace: Record<
        string,
        Array<{ x: number; y: { passed: number; total: number } }>
      > = {};
      for (const namespaceId of input.namespaceIds) {
        const rl = findNamespace(namespaceId);
        if (!rl) {
          continue;
        }
        const perBucketBase = rl.checks["24h"] * (stepMs / DAY_MS);
        const blockedShare = rl.blockedPct / 100;
        const points = [];
        for (let x = Math.floor(startTime / stepMs) * stepMs; x <= endTime; x += stepMs) {
          const rand = mulberry32(hashCode(`${rl.id}:${x}`));
          const total = Math.max(
            0,
            Math.round(perBucketBase * curveAt(rl.spark["24h"], x) * (0.7 + rand() * 0.6)),
          );
          const blocked = Math.min(total, Math.round(total * blockedShare * (0.5 + rand() * 1.2)));
          points.push({ x, y: { passed: total - blocked, total } });
        }
        timeseriesByNamespace[namespaceId] = points;
      }
      return {
        timeseriesByNamespace,
        granularity,
      } satisfies Outputs["ratelimit"]["logs"]["queryRatelimitTimeseriesBatch"];
    },

    "ratelimit.overview.logs.queryRatelimitLatencyTimeseries": (rawInput) => {
      const input = rawInput as {
        startTime: number;
        endTime: number;
        since: string;
        namespaceId: string;
      };
      const rl = findNamespace(input.namespaceId);
      if (!rl) {
        return {
          timeseries: [],
          granularity: "perMinute",
        } satisfies Outputs["ratelimit"]["overview"]["logs"]["queryRatelimitLatencyTimeseries"];
      }
      const { startTime, endTime } = resolveWindow(input);
      const { granularity, stepMs } = pickGranularity(endTime - startTime);
      const timeseries = [];
      for (let x = Math.floor(startTime / stepMs) * stepMs; x <= endTime; x += stepMs) {
        const rand = mulberry32(hashCode(`${rl.id}:lat:${x}`));
        const avg = Math.round((6 + rand() * 14) * 10) / 10;
        timeseries.push({
          x,
          y: { avg_latency: avg, p99_latency: Math.round(avg * (2.5 + rand() * 2) * 10) / 10 },
        });
      }
      return {
        timeseries,
        granularity,
      } satisfies Outputs["ratelimit"]["overview"]["logs"]["queryRatelimitLatencyTimeseries"];
    },

    "ratelimit.overview.logs.query": (rawInput) => {
      const input = rawInput as { namespaceId: string };
      const rl = findNamespace(input.namespaceId);
      if (!rl) {
        return {
          ratelimitOverviewLogs: [],
          total: 0,
        } satisfies Outputs["ratelimit"]["overview"]["logs"]["query"];
      }
      const identifiers = namespaceIdentifiers(rl);
      const rand = mulberry32(hashCode(`ovrows:${rl.id}`));
      const weights = identifiers.map(() => rand() ** 2 + 0.05);
      const weightSum = weights.reduce((a, b) => a + b, 0);
      const rows = identifiers.map((identifier, i) => {
        const total = Math.round((rl.checks["24h"] * weights[i]) / weightSum);
        const blocked = Math.round(total * (rl.blockedPct / 100) * (0.4 + rand() * 1.4));
        const passed = Math.max(0, total - blocked);
        return {
          time: Date.now() - Math.floor(rand() * 12 * HOUR_MS),
          identifier,
          request_id: `req_${Math.floor(rand() * 0xffffffff).toString(16)}`,
          passed_count: passed,
          blocked_count: blocked,
          passed_tokens: passed,
          total_tokens: total,
          override: null,
        };
      });
      rows.sort((a, b) => b.time - a.time);
      return {
        ratelimitOverviewLogs: rows,
        total: rows.length,
      } satisfies Outputs["ratelimit"]["overview"]["logs"]["query"];
    },

    "ratelimit.logs.query": (rawInput) => {
      const input = rawInput as { namespaceId: string; limit: number; page?: number };
      const rl = findNamespace(input.namespaceId);
      if (!rl) {
        return {
          ratelimitLogs: [],
          total: 0,
        } satisfies Outputs["ratelimit"]["logs"]["query"];
      }
      const identifiers = namespaceIdentifiers(rl);
      const page = input.page ?? 1;
      const limit = input.limit ?? 50;
      const totalRows = 160;
      const start = (page - 1) * limit;
      const rows = [];
      for (let i = start; i < Math.min(start + limit, totalRows); i++) {
        const rand = mulberry32(hashCode(`${rl.id}:log:${i}`));
        rows.push({
          request_id: `req_${Math.floor(rand() * 0xffffffff).toString(16)}${i}`,
          time: Date.now() - Math.floor(i * 4 * 60 * 1000 + rand() * 3 * 60 * 1000),
          identifier: identifiers[Math.floor(rand() * identifiers.length)],
          status: rand() < rl.blockedPct / 100 ? 0 : 1,
        });
      }
      return {
        ratelimitLogs: rows,
        total: totalRows,
      } satisfies Outputs["ratelimit"]["logs"]["query"];
    },

    "ratelimit.logs.enrichment": (rawInput) => {
      const input = rawInput as { requestIds: string[] };
      return {
        enrichment: input.requestIds.map((requestId) => {
          const rand = mulberry32(hashCode(`enrich:${requestId}`));
          const passed = rand() > 0.05;
          return {
            request_id: requestId,
            host: "api.acme.dev",
            method: "POST",
            path: "/v2/ratelimit.limit",
            request_headers: ["content-type: application/json"],
            request_body: '{"identifier":"…","limit":100,"duration":60000}',
            response_status: passed ? 200 : 429,
            response_headers: ["content-type: application/json"],
            response_body: `{"success":${passed}}`,
            service_latency: Math.round(3 + rand() * 25),
            user_agent: "unkey-go/0.5.1",
            region: "us-east-1",
          };
        }),
      } satisfies Outputs["ratelimit"]["logs"]["enrichment"];
    },

    "ratelimit.namespace.queryRatelimitLastUsed": (rawInput) => {
      const input = rawInput as { namespaceId: string; identifier: string };
      const rand = mulberry32(hashCode(`lastused:${input.namespaceId}:${input.identifier}`));
      return {
        identifier: input.identifier,
        lastUsed: Date.now() - Math.floor(rand() * 6 * HOUR_MS),
      } satisfies Outputs["ratelimit"]["namespace"]["queryRatelimitLastUsed"];
    },
  },

  merge: {
    "deploy.project.list": (_input, real) => {
      const realProjects = Array.isArray(real)
        ? (real as Outputs["deploy"]["project"]["list"])
        : [];
      return [...fakeProjects(), ...realProjects];
    },

    "ratelimit.namespace.list": (_input, real) => {
      const realNamespaces = Array.isArray(real)
        ? (real as Outputs["ratelimit"]["namespace"]["list"])
        : [];
      const fakes = activeWorld().ratelimits.map((rl) => ({ id: rl.id, name: rl.name }));
      return [...fakes, ...realNamespaces];
    },
  },
};

function notFoundError(message: string): Error {
  const err = new Error(message);
  err.name = "TRPCClientError";
  return err;
}
