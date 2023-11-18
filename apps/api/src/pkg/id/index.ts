import { encode } from "@/pkg/encoding/base58";

const namespaces = {
  workspace: "ws",
  api: "api",
  rootKey: "unkey",
  key: "key",
  keyAuth: "keyAuth", // not customer facing
  request: "req",
  test: "test", // not used in production but to mark test data
} as const;

export function newId(namespace: keyof typeof namespaces, byteLength = 8): string {
  const buf = new Uint8Array(byteLength);
  crypto.getRandomValues(buf);
  return [namespaces[namespace], encode(buf)].join("_");
}
