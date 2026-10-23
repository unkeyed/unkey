import {
  APP_SLUG_LENGTH,
  type App,
  type AppDeploymentRefs,
  createAppRequestSchema,
  latestDeploymentId,
} from "@/lib/collections/deploy/apps";
import { sanitizeImageRef, validateImageRef } from "@/lib/docker-image-ref";
import { slugify } from "@/lib/slugify";

const fallbackName = "app";

export const provisionalAppName = "new-app";

export const APP_NAME_TAKEN = "Another app in this project already has this name";

function toName(raw: string): string {
  const name = slugify(raw.replace(/[._]+/g, "-")) || fallbackName;
  return name.length < APP_SLUG_LENGTH.min ? `${name}-app` : name;
}

export function appNameFromRepo(repositoryFullName: string): string {
  return toName(repositoryFullName.split("/").pop() ?? "");
}

export function appNameFromImage(imageReference: string): string {
  const ref = validateImageRef(sanitizeImageRef(imageReference));
  return toName(ref.ok ? (ref.parts.path.split("/").pop() ?? "") : "");
}

export function uniqueAppName(base: string, takenSlugs: ReadonlySet<string>): string {
  if (!takenSlugs.has(base)) {
    return base;
  }
  let suffix = 2;
  while (takenSlugs.has(`${base}-${suffix}`)) {
    suffix++;
  }
  return `${base}-${suffix}`;
}

type PlaceholderCandidate = Pick<App, "id" | "slug" | "sourceType" | "repositoryFullName"> &
  AppDeploymentRefs;

function inNameFamily(slug: string, base: string): boolean {
  return slug === base || new RegExp(`^${base}-\\d+$`).test(slug);
}

export function hasGeneratedName(app: Pick<App, "slug" | "repositoryFullName">): boolean {
  return (
    app.repositoryFullName !== null &&
    inNameFamily(app.slug, appNameFromRepo(app.repositoryFullName))
  );
}

export function isPlaceholderApp(app: PlaceholderCandidate): boolean {
  return (
    inNameFamily(app.slug, provisionalAppName) &&
    app.sourceType === "git" &&
    app.repositoryFullName === null &&
    latestDeploymentId(app) === null
  );
}

export function findReusablePlaceholder(apps: readonly PlaceholderCandidate[]): string | null {
  return apps.find(isPlaceholderApp)?.id ?? null;
}

export function appNameError(name: string, takenSlugs: ReadonlySet<string>): string | null {
  const slug = slugify(name);
  const parsed = createAppRequestSchema.shape.slug.safeParse(slug);
  if (!parsed.success) {
    return parsed.error.issues[0]?.message ?? null;
  }
  return takenSlugs.has(slug) ? APP_NAME_TAKEN : null;
}
