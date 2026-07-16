import { customAlphabet } from "nanoid";

const nanoid = customAlphabet("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz");

const prefixes = {
  key: "key",
  policy: "pol",
  api: "api",
  request: "req",
  workspace: "ws",
  keyAuth: "ks", // keyspace
  role: "role",
  test: "test", // for tests only
  ratelimitNamespace: "rlns",
  ratelimitOverride: "rlor",
  permission: "perm",
  secret: "sec",
  headerRewrite: "hrw",
  sentinel: "gw",
  llmSentinel: "lgw",
  webhook: "wh",
  event: "evt",
  reporter: "rep",
  webhookDelivery: "whd",
  identity: "id",
  ratelimit: "rl",
  auditLog: "log",
  // correlation groups audit events emitted by one logical user action
  // so the dashboard can drill from any one event to the rest. Mirrors
  // pkg/uid.CorrelationPrefix in the Go side.
  correlation: "cor",
  fake: "fake",
  app: "app",
  environment: "env",
  environmentVariable: "evr",
  project: "proj",
  autoscalingPolicy: "asp",
} as const;

export function newId<TPrefix extends keyof typeof prefixes>(prefix: TPrefix) {
  return `${prefixes[prefix]}_${nanoid(12)}` as const;
}

// Mirrors pkg/uid/new.go: 9 chars from the full alphanumeric alphabet. Use for
// ids that must be indistinguishable from Go-generated ones, e.g.
// policy ids, which both the dashboard and svc/api write into the same
// sentinel_config blob.
const goUid = customAlphabet("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789");

export function newUid<TPrefix extends keyof typeof prefixes>(prefix: TPrefix) {
  return `${prefixes[prefix]}_${goUid(9)}` as const;
}

const dns1035Alpha = "abcdefghijklmnopqrstuvwxyz";

const dns = customAlphabet(`${dns1035Alpha}0123456789`);

export function dns1035(length?: number): string {
  const first = dns1035Alpha[Math.floor(Math.random() * dns1035Alpha.length)];
  const rest = dns(length ? length - 1 : undefined);

  return `${first}${rest}`;
}
