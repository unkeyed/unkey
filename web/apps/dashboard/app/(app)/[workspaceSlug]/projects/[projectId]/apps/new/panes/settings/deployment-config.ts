import { dockerContextSchema } from "@/app/(app)/[workspaceSlug]/projects/_components/repo-tree";
import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { z } from "zod";
import type { SourceKind } from "../../wizard-model";

export const deploymentConfigSchema = z.object({
  dockerContext: dockerContextSchema,
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
  { main: readonly SettingField[]; advanced: readonly SettingField[] }
> = {
  git: {
    main: ["dockerContext", "regions", "size"],
    advanced: ["port", "dockerfile", "buildCommand", "startCommand"],
  },
  oci: { main: ["port", "regions", "size"], advanced: [] },
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

export function directoryLabel(path: string): string {
  return path === "." || path === "" ? "./" : path;
}

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
