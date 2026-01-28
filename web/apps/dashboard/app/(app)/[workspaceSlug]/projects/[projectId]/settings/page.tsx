"use client";

import { useProject } from "../layout-provider";
import { GitHubSettingsClient } from "./components/github-settings-client";

export default function SettingsPage() {
  const { projectId } = useProject();

  return <GitHubSettingsClient projectId={projectId} />;
}
