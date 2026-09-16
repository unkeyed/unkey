"use client";
import { IconChartActivity2Outline12 } from "@unkey/icons";
// biome-ignore lint/correctness/noUnusedImports: React is needed for JSX
import React from "react";
import { BadgeTimestampCell } from "./badge-timestamp-cell";

export interface LastUpdatedCellProps {
  isSelected: boolean;
  lastUpdated?: number | null;
}

export const LastUpdatedCell = ({ isSelected, lastUpdated }: LastUpdatedCellProps) => (
  <BadgeTimestampCell
    isSelected={isSelected}
    timestamp={lastUpdated}
    icon={<IconChartActivity2Outline12 />}
    emptyText="Never used"
  />
);
