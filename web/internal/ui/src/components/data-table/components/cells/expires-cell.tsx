"use client";
import { Clock } from "@unkey/icons";
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
    icon={<Clock iconSize="sm-regular" />}
    emptyText="Never"
  />
);
