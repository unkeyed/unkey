"use client";

import { SettingsPageSkeleton } from "@unkey/ui";

export const SettingsClientSkeleton = () => (
  <SettingsPageSkeleton
    groups={[
      { title: "General", rows: [{ descriptionLines: 2 }, {}, {}] },
      { title: "Key defaults", rows: [{ descriptionLines: 2 }, { descriptionLines: 2 }] },
    ]}
    dangerZone={["w-16", "w-32"]}
  />
);
