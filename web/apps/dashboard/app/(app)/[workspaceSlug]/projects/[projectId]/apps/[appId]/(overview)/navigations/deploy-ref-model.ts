import type { Environment } from "@/lib/collections/deploy/environments";

export type TargetEnvironment = Pick<Environment, "id" | "slug" | "kind">;

export function environmentForRef(
  ref: string,
  productionBranch: string,
  environments: readonly TargetEnvironment[],
): string | null {
  if (!ref.trim()) {
    return null;
  }
  const kind = ref.trim() === productionBranch ? "production" : "preview";
  return environments.find((env) => env.kind === kind)?.slug ?? null;
}
