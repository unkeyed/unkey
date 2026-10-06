import { slugify } from "@/lib/slugify";

const fallbackName = "app";

export const provisionalAppName = "new-app";

// The API needs slugs of at least three characters.
const MIN_SLUG_LENGTH = 3;

function toName(raw: string): string {
  const name = slugify(raw.replace(/[._]+/g, "-")) || fallbackName;
  return name.length < MIN_SLUG_LENGTH ? `${name}-app` : name;
}

export function appNameFromRepo(repositoryFullName: string): string {
  return toName(repositoryFullName.split("/").pop() ?? "");
}

export function appNameFromImage(imageReference: string): string {
  const withoutDigest = imageReference.split("@")[0] ?? "";
  const lastSegment = withoutDigest.split("/").pop() ?? "";
  const withoutTag = lastSegment.split(":")[0] ?? "";
  return toName(withoutTag);
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

type PlaceholderCandidate = {
  id: string;
  slug: string;
  sourceType: "git" | "oci" | "unknown";
  repositoryFullName: string | null;
  currentDeploymentId: string | null;
  headlineDeployment: unknown;
};

const placeholderSlug = new RegExp(`^${provisionalAppName}(-\\d+)?$`);

export function findReusablePlaceholder(apps: readonly PlaceholderCandidate[]): string | null {
  const placeholder = apps.find(
    (app) =>
      placeholderSlug.test(app.slug) &&
      app.sourceType === "git" &&
      app.repositoryFullName === null &&
      app.currentDeploymentId === null &&
      app.headlineDeployment === null,
  );
  return placeholder?.id ?? null;
}
