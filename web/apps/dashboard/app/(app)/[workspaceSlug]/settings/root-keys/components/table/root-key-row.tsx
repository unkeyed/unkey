"use client";

import { PermissionsCell } from "@/components/root-keys-table/components/settings-root-keys/permissions-cell";
import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { RowActionSkeleton } from "@unkey/ui";
import { ResourceListItem, TimestampInfo } from "@unkey/ui";
import dynamic from "next/dynamic";

const RootKeysTableActions = dynamic(
  () =>
    import(
      "@/components/root-keys-table/components/settings-root-keys/root-keys-table-action.popover"
    ).then((mod) => mod.RootKeysTableActions),
  {
    loading: () => <RowActionSkeleton />,
    ssr: false,
  },
);

type RootKeyRowProps = {
  rootKey: RootKey;
  onSelect: (rootKey: RootKey) => void;
  onEditKey: (rootKey: RootKey) => void;
};

export function RootKeyRow({ rootKey, onSelect, onEditKey }: RootKeyRowProps) {
  return (
    <ResourceListItem className="flex flex-col gap-3 px-4 py-3 transition-colors hover:bg-grayA-2 md:flex-row md:items-center md:gap-0">
      <button
        type="button"
        onClick={() => onSelect(rootKey)}
        className="absolute inset-0 z-10 cursor-pointer"
        aria-label={`Edit ${rootKey.name ?? "unnamed root key"}`}
      />

      <div className="min-w-0 md:w-[35%] md:shrink-0">
        <span className="truncate text-[13px] font-medium text-accent-12">
          {rootKey.name ?? "Unnamed root key"}
        </span>
      </div>

      <div className="min-w-0 md:w-[35%] md:shrink-0">
        <PermissionsCell permissions={rootKey.permissions} isSelected={false} />
      </div>

      <div className="flex items-center gap-4 md:w-[30%] md:shrink-0 md:justify-end">
        <span className="relative z-20">
          <TimestampInfo
            value={rootKey.createdAt}
            displayType="relative"
            side="left"
            align="center"
            className="text-[13px] text-gray-9"
          />
        </span>
        <div className="relative z-20" role="presentation">
          <RootKeysTableActions rootKey={rootKey} onEditKey={onEditKey} />
        </div>
      </div>
    </ResourceListItem>
  );
}
