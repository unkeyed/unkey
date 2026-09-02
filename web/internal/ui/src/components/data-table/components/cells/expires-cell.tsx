"use client";
import { IconClockOutline18 } from "nucleo-ui-outline-18";
// biome-ignore lint/correctness/noUnusedImports: React is needed for JSX
import React from "react";
import { BadgeTimestampCell } from "./badge-timestamp-cell";

export interface ExpiresCellProps {
  isSelected: boolean;
  expiresAt?: number | null;
}

export const ExpiresCell = ({ isSelected, expiresAt }: ExpiresCellProps) => (
  <BadgeTimestampCell
    isSelected={isSelected}
    timestamp={expiresAt}
    icon={<IconClockOutline18 />}
    emptyText="Never"
  />
);
