import { customAlphabet } from "nanoid";

const nanoid = customAlphabet("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz");

const prefixes = {
  key: "key",
  policy: "pol",
  api: "api",
  request: "req",
  workspace: "ws",
  keyAuth: "ks", // keyspace
  vercelBinding: "vb",
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
  auditLogBucket: "buk",
  auditLog: "log",
  fake: "fake",
  environment: "env",
  environmentVariable: "evr",
  project: "proj",
} as const;

export function newId<TPrefix extends keyof typeof prefixes>(prefix: TPrefix) {
  return `${prefixes[prefix]}_${nanoid(12)}` as const;
}

const dns1035Alpha = "abcdefghijklmnopqrstuvwxyz";

const dns = customAlphabet(`${dns1035Alpha}0123456789`);

export function dns1035(length?: number): string {
  const first = dns1035Alpha[Math.floor(Math.random() * dns1035Alpha.length)];
  const rest = dns(length ? length - 1 : undefined);

  return `${first}${rest}`;
}
