"use client";

import type * as React from "react";
import { SettingsDangerZone, SettingsZoneRowSkeleton } from "./settings-card";
import {
  SettingsForm,
  SettingsGroup,
  SettingsGroupContent,
  SettingsGroupTitle,
  SettingsRowSkeleton,
} from "./settings-row";

type SettingsSkeletonRow = {
  descriptionLines?: 0 | 1 | 2;
  control?: React.ReactNode;
};

type SettingsSkeletonGroup = {
  title?: string;
  rows: readonly SettingsSkeletonRow[];
};

const NOT_SAVABLE = { status: "disabled" } as const;

function SettingsPageSkeleton({
  groups,
  dangerZone,
}: {
  groups: readonly SettingsSkeletonGroup[];
  dangerZone?: readonly string[];
}) {
  return (
    <div aria-busy="true" className="contents">
      <output className="sr-only">Loading settings…</output>
      {groups.map((group, groupIndex) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: skeleton groups are static and never reorder
        <SettingsGroup key={groupIndex}>
          {group.title ? <SettingsGroupTitle>{group.title}</SettingsGroupTitle> : null}
          <SettingsGroupContent>
            {/* A disabled form makes the group render its Save footer, as the loaded page does. */}
            <SettingsForm
              dirty={false}
              saveState={NOT_SAVABLE}
              onSubmit={(e) => e.preventDefault()}
            >
              {group.rows.map((row, rowIndex) => (
                // biome-ignore lint/suspicious/noArrayIndexKey: skeleton rows are static and never reorder
                <SettingsRowSkeleton key={rowIndex} {...row} />
              ))}
            </SettingsForm>
          </SettingsGroupContent>
        </SettingsGroup>
      ))}
      {dangerZone ? (
        <SettingsDangerZone>
          {dangerZone.map((actionClassName, rowIndex) => (
            // biome-ignore lint/suspicious/noArrayIndexKey: skeleton rows are static and never reorder
            <SettingsZoneRowSkeleton key={rowIndex} actionClassName={actionClassName} />
          ))}
        </SettingsDangerZone>
      ) : null}
    </div>
  );
}

SettingsPageSkeleton.displayName = "SettingsPageSkeleton";

export { SettingsPageSkeleton, type SettingsSkeletonGroup, type SettingsSkeletonRow };
