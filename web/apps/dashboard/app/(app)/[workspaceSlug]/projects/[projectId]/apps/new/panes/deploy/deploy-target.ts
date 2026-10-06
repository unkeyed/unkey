import type { EnvironmentKind } from "@/lib/collections/deploy/environments";
import type { SourceKind } from "../../wizard-model";

type TargetEnvironment = { slug: string; kind: EnvironmentKind };

// Matches the flows this wizard replaced: a repository's first deploy goes to
// production, an image's first deploy goes to preview.
const firstDeployKind: Record<SourceKind, EnvironmentKind> = {
  git: "production",
  oci: "preview",
};

export function firstDeployEnvironment(
  source: SourceKind,
  environments: readonly TargetEnvironment[],
): string | null {
  const preferred = environments.find((env) => env.kind === firstDeployKind[source]);
  if (preferred) {
    return preferred.slug;
  }
  return source === "oci" ? (environments.at(0)?.slug ?? null) : null;
}
