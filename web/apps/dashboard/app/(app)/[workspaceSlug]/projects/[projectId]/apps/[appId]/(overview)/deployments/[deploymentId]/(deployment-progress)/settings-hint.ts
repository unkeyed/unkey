import type { AppSettingsTarget } from "@/lib/navigation/routes/projects";

type SettingsHint = "build" | "environment";

const HINT_TARGET: Record<
  SettingsHint,
  (environmentSlug: string | undefined) => AppSettingsTarget
> = {
  build: () => ({ page: "build" }),
  environment: (environment) => ({ page: "environments", environment }),
};

const SETTINGS_HINTS: ReadonlyArray<{ pattern: string; hint: SettingsHint }> = [
  // Dockerfile path / docker context (from build.go extractUserBuildError)
  { pattern: "check that the file path is correct", hint: "build" },
  { pattern: "dockerfile appears to be empty", hint: "build" },
  { pattern: "build target stage was not found", hint: "build" },
  { pattern: "check the root directory", hint: "build" },
  // Region configuration (from deploy_handler.go createTopologies)
  { pattern: "configure at least one region", hint: "environment" },
  // Git branch (from deploy_handler.go buildImage)
  { pattern: "git branch could not be resolved", hint: "build" },
];

export function settingsTargetFor(
  error: string,
  environmentSlug: string | undefined,
): AppSettingsTarget | undefined {
  const lower = error.toLowerCase();
  const found = SETTINGS_HINTS.find(({ pattern }) => lower.includes(pattern));
  return found ? HINT_TARGET[found.hint](environmentSlug) : undefined;
}
