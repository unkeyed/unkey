import { APP_SETTINGS_PAGES, type AppSettingsPage } from "@/lib/navigation/routes/projects";

export type SettingsSection = "general" | AppSettingsPage;

export const SETTINGS_RAIL: readonly SettingsSection[] = ["general", ...APP_SETTINGS_PAGES];

export const SETTINGS_SECTIONS: Record<SettingsSection, { label: string; description?: string }> = {
  general: { label: "General" },
  build: { label: "Build" },
  runtime: { label: "Runtime" },
  advanced: { label: "Advanced" },
};

export function activeSection(segments: string[]): SettingsSection {
  return APP_SETTINGS_PAGES.find((page) => segments.includes(page)) ?? "general";
}

export function sectionPage(section: SettingsSection): AppSettingsPage | undefined {
  return section === "general" ? undefined : section;
}

type SettingsHeader = { back?: AppSettingsPage; title: string; description?: string };

export function settingsHeader(section: SettingsSection): SettingsHeader {
  const { label, description } = SETTINGS_SECTIONS[section];
  return { title: label, description };
}
