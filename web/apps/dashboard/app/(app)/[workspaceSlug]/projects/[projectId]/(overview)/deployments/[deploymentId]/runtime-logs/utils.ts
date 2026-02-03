import type { RuntimeLog } from "./types";

export function safeParseAttributes(log: RuntimeLog): Record<string, unknown> | null {
  if (!log.attributes) {
    return null;
  }
  if (typeof log.attributes === "object") {
    return log.attributes;
  }
  return null;
}
