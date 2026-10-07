import type { AppSettingsPage } from "@/lib/navigation/routes/projects";

export const SETTINGS_SECTIONS = [
  { label: "General", page: undefined },
  { label: "Build", page: "build" },
  { label: "Runtime", page: "runtime" },
  {
    label: "Environments",
    page: "environments",
    description: "Configure how each environment deploys and runs.",
  },
  { label: "Advanced", page: "advanced" },
] as const satisfies ReadonlyArray<{
  label: string;
  page: AppSettingsPage | undefined;
  description?: string;
}>;

export function activeSection(segment: string | null) {
  return SETTINGS_SECTIONS.find((section) => section.page === segment) ?? SETTINGS_SECTIONS[0];
}
