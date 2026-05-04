"use client";
import { ChartActivity2 } from "@unkey/icons";
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
    icon={<ChartActivity2 iconSize="sm-regular" />}
    emptyText="Never used"
  />
);
