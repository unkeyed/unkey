"use client";
import { VerificationBarChart } from "@/components/api-keys-table/components/bar-chart";
import { LastUsedCell } from "@/components/api-keys-table/components/last-used";
import { StatusDisplay } from "@/components/api-keys-table/components/status-cell";
import { shortenId } from "@/lib/shorten-id";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { cn } from "@/lib/utils";
import { Focus, Key } from "@unkey/icons";
import type { DataTableColumnDef } from "@unkey/ui";
import {
  Checkbox,
  HiddenValueCell,
  InfoTooltip,
  Loading,
  RowActionSkeleton,
  SortableHeader,
} from "@unkey/ui";
import dynamic from "next/dynamic";
import Link from "next/link";
import { useState } from "react";

const KeysTableActionPopover = dynamic(
  () =>
    import(
      "@/components/api-keys-table/components/actions/keys-table-action.popover.constants"
    ).then((mod) => mod.KeysTableActions),
  {
    ssr: false,
    loading: () => <RowActionSkeleton />,
  },
);

export const API_KEY_COLUMN_IDS = {
  KEY: "key",
  VALUE: "value",
  USAGE: "usage",
  LAST_USED: "last_used",
  STATUS: "status",
  ACTION: "action",
} as const;

type CreateApiKeyColumnsOptions = {
  keyspaceId: string;
  apiId: string;
  workspaceSlug: string;
  selectedKeyId: string | null;
  navigatingKeyId: string | null;
  selectedKeys: Set<string>;
  onToggleSelection: (keyId: string) => void;
  onNavigateToKey: (keyId: string) => void;
};

type KeyIdCellProps = {
  keyData: KeyDetails;
  keyspaceId: string;
  apiId: string;
  workspaceSlug: string;
  isNavigating: boolean;
  selectedKeys: Set<string>;
  onToggleSelection: (keyId: string) => void;
  onNavigate: (keyId: string) => void;
};

const KeyIdCell = ({
  keyData,
  keyspaceId,
  apiId,
  workspaceSlug,
  isNavigating,
  selectedKeys,
  onToggleSelection,
  onNavigate,
}: KeyIdCellProps) => {
  const [isHovered, setIsHovered] = useState(false);
  const identity = keyData.identity?.external_id ?? keyData.owner_id;
  const isKeySelected = selectedKeys.has(keyData.id);

  const iconContainer = (
    <div
      className={cn(
        "size-5 rounded-sm flex items-center justify-center cursor-pointer relative",
        identity ? "bg-successA-3" : "bg-grayA-3",
        isKeySelected && "bg-brand-5",
      )}
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
    >
      {isNavigating ? (
        <div className={cn(identity ? "text-successA-11" : "text-grayA-11")}>
          <Loading size={18} />
        </div>
      ) : (
        <>
          <div
            className={cn(
              isKeySelected || isHovered ? "opacity-0 pointer-events-none" : "opacity-100",
            )}
          >
            {identity ? (
              <Focus iconSize="md-medium" className="text-successA-11" />
            ) : (
              <Key iconSize="md-medium" />
            )}
          </div>
          <Checkbox
            checked={isKeySelected}
            className={cn(
              "size-4 [&_svg]:size-3 absolute",
              isKeySelected || isHovered
                ? "opacity-100"
                : "opacity-0 pointer-events-none focus-visible:opacity-100 focus-visible:pointer-events-auto",
            )}
            onCheckedChange={() => onToggleSelection(keyData.id)}
          />
        </>
      )}
    </div>
  );

  return (
    <div className="flex flex-col items-start px-4.5 py-1.5">
      <div className="flex gap-4 items-center">
        {identity ? (
          <InfoTooltip
            delayDuration={100}
            variant="muted"
            position={{ side: "right" }}
            className="bg-gray-1 px-4 py-2 border border-gray-4 shadow-md font-medium text-xs text-accent-12"
            content={
              <>
                This key is associated with the identity:{" "}
                {keyData.identity_id && workspaceSlug ? (
                  <Link
                    title="View details for identity"
                    className="font-mono group-hover:underline decoration-dotted"
                    href={`/${workspaceSlug}/identities/${keyData.identity_id}`}
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    <span className="font-mono bg-gray-4 p-1 rounded-sm">{identity}</span>
                  </Link>
                ) : (
                  <span className="font-mono bg-gray-4 p-1 rounded-sm">{identity}</span>
                )}
              </>
            }
            asChild
          >
            {iconContainer}
          </InfoTooltip>
        ) : (
          iconContainer
        )}

        <div className="flex flex-col gap-1 text-xs">
          <Link
            title={`View details for ${keyData.id}`}
            className="font-mono group-hover:underline decoration-dotted"
            href={`/${workspaceSlug}/apis/${apiId}/keys/${keyspaceId}/${keyData.id}`}
            onClick={() => {
              onNavigate(keyData.id);
            }}
          >
            <div className="font-mono font-medium truncate text-brand-12">
              {shortenId(keyData.id)}
            </div>
          </Link>
          {keyData.name && (
            <span className="font-sans text-accent-9 truncate max-w-30" title={keyData.name}>
              {keyData.name}
            </span>
          )}
        </div>
      </div>
    </div>
  );
};

export const createApiKeyColumns = ({
  keyspaceId,
  apiId,
  workspaceSlug,
  selectedKeyId,
  navigatingKeyId,
  selectedKeys,
  onToggleSelection,
  onNavigateToKey,
}: CreateApiKeyColumnsOptions): DataTableColumnDef<KeyDetails>[] => [
  {
    id: API_KEY_COLUMN_IDS.KEY,
    accessorKey: "id",
    sortDescFirst: true,
    header: ({ header }) => (
      <SortableHeader key={API_KEY_COLUMN_IDS.KEY} header={header}>
        Key
      </SortableHeader>
    ),
    meta: {
      width: "20%",
      headerClassName: "pl-[18px]",
    },
    cell: ({ row }) => {
      const key = row.original;
      return (
        <KeyIdCell
          keyData={key}
          keyspaceId={keyspaceId}
          apiId={apiId}
          workspaceSlug={workspaceSlug}
          isNavigating={key.id === navigatingKeyId}
          selectedKeys={selectedKeys}
          onToggleSelection={onToggleSelection}
          onNavigate={onNavigateToKey}
        />
      );
    },
  },
  {
    id: API_KEY_COLUMN_IDS.VALUE,
    accessorKey: "start",
    header: ({ header }) => (
      <SortableHeader key={API_KEY_COLUMN_IDS.VALUE} header={header}>
        Value
      </SortableHeader>
    ),
    meta: {
      width: "15%",
    },
    cell: ({ row }) => {
      const key = row.original;
      return (
        <HiddenValueCell value={key.start} title="Value" selected={key.id === selectedKeyId} />
      );
    },
  },
  {
    id: API_KEY_COLUMN_IDS.USAGE,
    header: "Usage in last 36h",
    enableSorting: false,
    meta: {
      width: "20%",
    },
    cell: ({ row }) => {
      const key = row.original;
      return (
        <VerificationBarChart
          keyAuthId={keyspaceId}
          keyId={key.id}
          selected={key.id === selectedKeyId}
        />
      );
    },
  },
  {
    id: API_KEY_COLUMN_IDS.LAST_USED,
    accessorKey: "last_used_at",
    sortDescFirst: true,
    header: ({ header }) => (
      <SortableHeader key={API_KEY_COLUMN_IDS.LAST_USED} header={header}>
        Last Used
      </SortableHeader>
    ),
    meta: {
      width: "15%",
    },
    cell: ({ row }) => {
      const key = row.original;
      return <LastUsedCell lastUsedAt={key.last_used_at} isSelected={key.id === selectedKeyId} />;
    },
  },
  {
    id: API_KEY_COLUMN_IDS.STATUS,
    header: "Status",
    enableSorting: false,
    meta: {
      width: "15%",
    },
    cell: ({ row }) => {
      const key = row.original;
      return (
        <StatusDisplay keyData={key} keyAuthId={keyspaceId} isSelected={key.id === selectedKeyId} />
      );
    },
  },
  {
    id: API_KEY_COLUMN_IDS.ACTION,
    header: "",
    enableSorting: false,
    meta: {
      width: "15%",
    },
    cell: ({ row }) => {
      const key = row.original;
      return <KeysTableActionPopover keyData={key} apiId={apiId} keyspaceId={keyspaceId} />;
    },
  },
];
