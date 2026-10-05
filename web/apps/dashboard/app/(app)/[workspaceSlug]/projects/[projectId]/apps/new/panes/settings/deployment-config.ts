import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { z } from "zod";
import type { SourceKind } from "../../wizard-model";

const dockerContextSegment = /^[A-Za-z0-9._-]+$/;

const rootDirectoryMarkers = new Set([
  "build.gradle",
  "build.gradle.kts",
  "cargo.toml",
  "composer.json",
  "gemfile",
  "go.mod",
  "mix.exs",
  "package.json",
  "pipfile",
  "pom.xml",
  "pyproject.toml",
  "requirements.txt",
]);

export const deploymentConfigSchema = z.object({
  dockerContext: z
    .string()
    .min(1, "Enter a root directory or use '.' for the repository root.")
    .refine(
      (path) =>
        path === "." ||
        (path === path.trim() &&
          !path.startsWith("/") &&
          !path.includes("\\") &&
          path
            .split("/")
            .every(
              (segment) =>
                segment !== "." && segment !== ".." && dockerContextSegment.test(segment),
            )),
      "Enter a path relative to the repository root, like api or services/api.",
    ),
  regions: z.array(z.string()).min(1, "Select at least one region"),
  port: z
    .number({ error: "Enter a port number" })
    .int()
    .min(1, "Enter a port from 1 to 65535")
    .max(65535, "Enter a port from 1 to 65535"),
  dockerfile: z.string(),
  buildCommand: z.string().max(1000),
  startCommand: z.string().max(1000),
  size: z.object({ cpuMillicores: z.number().int(), memoryMib: z.number().int() }),
});

export type DeploymentConfig = z.infer<typeof deploymentConfigSchema>;

export type SettingField = keyof DeploymentConfig;

export const settingsLayout: Record<
  SourceKind,
  { main: readonly SettingField[]; advanced: readonly SettingField[]; usesRepoTree: boolean }
> = {
  git: {
    main: ["dockerContext", "regions", "size"],
    advanced: ["port", "dockerfile", "buildCommand", "startCommand"],
    usesRepoTree: true,
  },
  oci: { main: ["port", "regions", "size"], advanced: [], usesRepoTree: false },
};

export const settingTitle: Record<SettingField, string> = {
  dockerContext: "Root directory",
  regions: "Regions",
  port: "Port",
  dockerfile: "Dockerfile",
  buildCommand: "Build command",
  startCommand: "Start command",
  size: "Size",
};

export type BuildMethod = "automatic" | "dockerfile";

export function resolveBuildMethod(dockerfile: string): BuildMethod {
  return dockerfile.trim() === "" ? "automatic" : "dockerfile";
}

export function readDeploymentConfig(settings: EnvironmentSettings): DeploymentConfig {
  return {
    dockerContext: settings.dockerContext,
    regions: settings.regions.map((region) => region.name),
    port: settings.port,
    dockerfile: settings.dockerfile,
    buildCommand: settings.buildCommand,
    startCommand: settings.command.join(" "),
    size: { cpuMillicores: settings.cpuMillicores, memoryMib: settings.memoryMib },
  };
}

export function applyDeploymentConfig(draft: EnvironmentSettings, config: DeploymentConfig): void {
  const replicasMin = draft.regions.at(0)?.replicasMin ?? 1;
  const replicasMax = draft.regions.at(0)?.replicasMax ?? 1;

  draft.dockerContext = config.dockerContext;
  draft.port = config.port;
  draft.cpuMillicores = config.size.cpuMillicores;
  draft.memoryMib = config.size.memoryMib;
  draft.dockerfile = config.dockerfile;
  draft.buildCommand = config.buildCommand;
  draft.command = config.startCommand.trim().split(/\s+/).filter(Boolean);
  draft.regions = config.regions.map(
    (name) =>
      draft.regions.find((region) => region.name === name) ?? { name, replicasMin, replicasMax },
  );
}

export type RepoTreeEntry = { path: string; type: string };

type RootDirectorySuggestion = { path: string; marker: string };

export function suggestRootDirectories(tree: RepoTreeEntry[]): RootDirectorySuggestion[] {
  const markersByPath = new Map<string, string>();
  for (const entry of tree) {
    if (entry.type !== "blob") {
      continue;
    }
    const fileName = entry.path.split("/").pop() ?? "";
    const normalized = fileName.toLowerCase();
    if (!rootDirectoryMarkers.has(normalized) && !normalized.includes("dockerfile")) {
      continue;
    }
    const separatorIndex = entry.path.lastIndexOf("/");
    const path = separatorIndex === -1 ? "." : entry.path.slice(0, separatorIndex);
    if (path !== "." && !markersByPath.has(path)) {
      markersByPath.set(path, fileName);
    }
  }

  return [
    { path: ".", marker: "Repository root" },
    ...Array.from(markersByPath, ([path, marker]) => ({ path, marker })).sort((a, b) =>
      a.path.localeCompare(b.path),
    ),
  ];
}

export function findDockerfiles(tree: RepoTreeEntry[], dockerContext: string): string[] {
  const prefix = dockerContext === "." ? "" : `${dockerContext}/`;
  return tree
    .filter((entry) => {
      const fileName = entry.path.split("/").pop() ?? "";
      return (
        entry.type === "blob" &&
        fileName.toLowerCase().includes("dockerfile") &&
        entry.path.startsWith(prefix)
      );
    })
    .map((entry) => entry.path.slice(prefix.length));
}
