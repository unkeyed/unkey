"use client";

import { SettingsPageSkeleton } from "@unkey/ui";

export const SettingsClientSkeleton = () => (
  <SettingsPageSkeleton groups={[{ rows: [{ descriptionLines: 2 }, {}] }]} dangerZone={["w-36"]} />
);
