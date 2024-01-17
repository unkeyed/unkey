import { customAlphabet } from "nanoid";
export const nanoid = customAlphabet("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz");

const prefixes = {
  key: "key",
  policy: "pol",
  api: "api",
  request: "req",
  workspace: "ws",
  keyAuth: "key_auth",
  vercelBinding: "vb",
  role: "role",
  test: "test", // for tests only
  auditLog: "log",
} as const;

export function newId<TPrefix extends keyof typeof prefixes>(prefix: TPrefix) {
  return `${prefixes[prefix]}_${nanoid(16)}` as const;
}
