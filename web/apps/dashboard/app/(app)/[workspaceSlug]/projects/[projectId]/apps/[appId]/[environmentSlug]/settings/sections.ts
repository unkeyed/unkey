import type { AppSettingsPage } from "@/lib/navigation/routes/projects";

export type SettingsScope = "environment" | "application";

export type SettingsSection = {
  label: string;
  scope: SettingsScope;
  page: AppSettingsPage | undefined;
  tone?: "danger";
};

export const SETTINGS_SECTIONS = [
  { label: "Compute", scope: "environment", page: undefined },
  { label: "Domains", scope: "environment", page: "domains" },
  { label: "Deploys", scope: "environment", page: "deploys" },
  { label: "Build", scope: "application", page: "build" },
  { label: "Runtime", scope: "application", page: "runtime" },
  { label: "Advanced", scope: "application", page: "advanced" },
  { label: "Danger zone", scope: "application", page: "danger", tone: "danger" },
] as const satisfies readonly SettingsSection[];

export const SETTINGS_GROUPS: ReadonlyArray<{ scope: SettingsScope; label: string }> = [
  { scope: "environment", label: "Environment" },
  { scope: "application", label: "Application" },
];
