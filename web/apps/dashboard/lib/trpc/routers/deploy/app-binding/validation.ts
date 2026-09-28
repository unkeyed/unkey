import { z } from "zod";

const privateZone = "unkey.internal";
const maxNameLength = 63;
const reservedPrefix = "unkey";
const namePattern = /^[a-z](?:[a-z0-9-]{0,61}[a-z0-9])?$/;

export const bindingEndpointsSchema = z
  .object({
    appId: z.string().min(1),
    environmentId: z.string().min(1),
    targetAppId: z.string().min(1),
  })
  .refine((value) => value.appId !== value.targetAppId, {
    message: "Choose another app. Connections within this app are automatic.",
    path: ["targetAppId"],
  });

export const bindingNameSchema = z
  .string()
  .trim()
  .min(1, "Name is required")
  .max(maxNameLength)
  .regex(namePattern, "Start with a letter and use lowercase letters, numbers, and hyphens")
  .refine((value) => !value.startsWith(reservedPrefix), "Names starting with unkey are reserved");

export function bindingHost(name: string): string {
  return `${name}.${privateZone}`;
}

export function bindingHostVariable(name: string): string {
  return `${name.replaceAll("-", "_").toUpperCase()}_HOST`;
}

export function defaultBindingName(
  slug: string,
  isTaken: (name: string) => boolean,
): string | undefined {
  const base = baseName(slug);
  for (let attempt = 1; attempt <= 20; attempt++) {
    const suffix = attempt === 1 ? "" : `-${attempt}`;
    const candidate = `${base.slice(0, maxNameLength - suffix.length).replace(/-+$/, "")}${suffix}`;
    if (bindingNameSchema.safeParse(candidate).success && !isTaken(candidate)) {
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
    /^[a-z]/.test(cleaned) && !cleaned.startsWith(reservedPrefix) ? cleaned : `app-${cleaned}`;
  return prefixed.slice(0, maxNameLength).replace(/-+$/, "");
}
