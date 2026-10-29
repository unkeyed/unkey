"use client";

import type { AppSettingsPage } from "@/lib/navigation/routes/projects";
import { notFound, useParams } from "next/navigation";
import type { ReactNode } from "react";
import { AdvancedSettings } from "../../../../../_components/settings/advanced-settings/advanced-settings";
import { BuildSettings } from "../../../../../_components/settings/build-settings/build-settings";

const PAGES: Record<Exclude<AppSettingsPage, "environments">, ReactNode> = {
  build: <BuildSettings />,
  advanced: <AdvancedSettings />,
};

export default function EnvironmentSettingsPage() {
  const { page } = useParams<{ page: string }>();
  const content = Object.entries(PAGES).find(([key]) => key === page)?.[1];
  if (!content) {
    notFound();
  }
  return content;
}
