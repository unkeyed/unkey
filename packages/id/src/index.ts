import baseX from "base-x";

const prefixes = {
  key: "key",
  policy: "pol",
  api: "api",
  request: "req",
  workspace: "ws",
  keyAuth: "key_auth",
} as const;

export function newId(prefix: keyof typeof prefixes): string {
  const buf = new Uint8Array(16);
  crypto.getRandomValues(buf);
  return [
    prefixes[prefix],
    baseX("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz").encode(buf),
  ].join("_");
}
