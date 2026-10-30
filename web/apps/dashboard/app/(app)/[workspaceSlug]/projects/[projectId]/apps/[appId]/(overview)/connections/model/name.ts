import { z } from "zod";
import type { ListedConnection } from "./target";

const MAX_NAME_LENGTH = 63;
const RESERVED_PREFIX = "unkey";

export const connectionNameSchema = z
  .string()
  .trim()
  .min(1, "Name is required")
  .max(MAX_NAME_LENGTH)
  .regex(
    /^[a-z](?:[a-z0-9-]{0,61}[a-z0-9])?$/,
    "Start with a letter and use lowercase letters, numbers, and hyphens",
  )
  .refine((value) => value !== "self", "The name self is reserved")
  .refine((value) => !value.startsWith(RESERVED_PREFIX), "Names starting with unkey are reserved");

export function defaultConnectionName(
  slug: string,
  isTaken: (name: string) => boolean,
): string | undefined {
  const base = baseName(slug);
  for (let attempt = 1; attempt <= 20; attempt++) {
    const suffix = attempt === 1 ? "" : `-${attempt}`;
    const candidate = `${base.slice(0, MAX_NAME_LENGTH - suffix.length).replace(/-+$/, "")}${suffix}`;
    if (connectionNameSchema.safeParse(candidate).success && !isTaken(candidate)) {
      return candidate;
    }
  }
  return undefined;
}

function baseName(slug: string): string {
  const cleaned = slug
    .toLowerCase()
    .replace(/[^a-z0-9-]+/g, "-")
    .replace(/-{2,}/g, "-")
    .replace(/^-+|-+$/g, "");
  const prefixed =
    /^[a-z]/.test(cleaned) && !cleaned.startsWith(RESERVED_PREFIX) ? cleaned : `app-${cleaned}`;
  return prefixed.slice(0, MAX_NAME_LENGTH).replace(/-+$/, "");
}

export function connectionHostVariable(name: string): string {
  return `${name.replaceAll("-", "_").toUpperCase()}_HOST`;
}

export function takenNamesOf(
  connections: readonly ListedConnection[],
  targetAppId: string,
): ReadonlySet<string> {
  return new Set(
    connections
      .filter((connection) => connection.targetAppId !== targetAppId)
      .map((connection) => connection.name),
  );
}
