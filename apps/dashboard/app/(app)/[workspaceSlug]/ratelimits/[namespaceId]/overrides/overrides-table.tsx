"use client";
import { VirtualTable } from "@/components/virtual-table";
import type { Column } from "@/components/virtual-table/types";
import { collection } from "@/lib/collections";
import { formatNumber } from "@/lib/fmt";
import { cn } from "@/lib/utils";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { Badge, CopyButton, Empty, InfoTooltip } from "@unkey/ui";
import ms from "ms";
import { useState } from "react";
import { IdentifierDialog } from "../_components/identifier-dialog";
import { LastUsedCell } from "./last-used-cell";
import { OverridesTableAction } from "./logs-actions";

type Override = {
  id: string;
  identifier: string;
  limit: number;
  duration: number;
  async: boolean | null;
};

type Props = {
  namespaceId: string;
};

const STATUS_STYLES = {
  default: {
    base: "text-accent-9",
    hover: "hover:text-accent-11 dark:hover:text-accent-12 hover:bg-accent-3",
    selected: "text-accent-11 bg-accent-3 dark:text-accent-12",
    badge: {
      default: "bg-accent-4 text-accent-11 group-hover:bg-accent-5",
      selected: "bg-accent-5 text-accent-12 hover:bg-hover-5",
    },
    focusRing: "focus:ring-accent-7",
  },
};

const getRowClassName = () => {
  const style = STATUS_STYLES.default;
  return cn(
    style.base,
    style.hover,
    "group rounded-md",
    "focus:outline-none focus:ring-1 focus:ring-opacity-40",
    style.focusRing,
  );
};

export const OverridesTable = ({ namespaceId }: Props) => {
  const [selectedOverride, setSelectedOverride] = useState<Override | null>(null);
  const [isDialogOpen, setIsDialogOpen] = useState(false);

  const { data: overrides, isLoading } = useLiveQuery((q) =>
    q
      .from({ override: collection.ratelimitOverrides })
      .where(({ override }) => eq(override.namespaceId, namespaceId)),
  );

  const handleRowClick = (override: Override) => {
    setSelectedOverride(override);
    setIsDialogOpen(true);
  };

  const columns: Column<Override>[] = [
    {
      key: "id",
      header: "ID",
      headerClassName: "pl-2",
      width: "20%",
      render: (override) => (
        <div className="pl-2">
          <InfoTooltip
            content={
              <div className="inline-flex justify-center gap-3 items-center font-mono text-xs text-gray-11">
                <span className="">{override.id}</span>
                <CopyButton value={override.id} />
              </div>
            }
            position={{ side: "bottom", align: "start" }}
          >
            <div className="flex flex-row justify-center items-center font-mono text-xs text-gray-11 truncate">
              {override.id}
            </div>
          </InfoTooltip>
        </div>
      ),
    },
    {
      key: "identifier",
      header: "Identifier",
      headerClassName: "pl-2",
      width: "auto",
      render: (override) => (
        <div className="inline-flex items-start pl-2">
          <InfoTooltip
            content={
              <div className="flex gap-3">
                <div className="flex justify-start items-center break-all max-w-[400px]">
                  {override.identifier}
                </div>
                <div className="flex flex-col justify-center items-center w-4">
                  <CopyButton value={override.identifier} />
                </div>
              </div>
            }
            position={{ side: "bottom", align: "start" }}
          >
            <pre className="text-[11px] text-gray-11 sm:max-w-[100px] md:max-w-[100px] lg:max-w-[320px] xl:max-w-[600px] truncate">
              {override.identifier}
            </pre>
          </InfoTooltip>
        </div>
      ),
    },
    {
      key: "limits",
      header: "Limits",
      width: "10%",
      render: (override) => (
        <div className="inline-grid grid-cols-[1fr_auto_1fr] items-center gap-2">
          <div className="flex justify-start">
            <Badge
              className={cn(
                " px-2 rounded-md font-mono truncate uppercase",
                STATUS_STYLES.default.badge.default,
              )}
            >
              {formatNumber(override.limit)}/{ms(override.duration)}
            </Badge>
          </div>
          {/*<span className="text-content-subtle">/</span>
          <div className="flex justify-start">
            <Badge
              className={cn(
                "uppercase px-2 rounded-md font-mono",
                STATUS_STYLES.default.badge.default,
              )}
            >
              {ms(override.duration)}
            </Badge>
          </div>*/}
        </div>
      ),
    },
    {
      key: "lastUsed",
      header: "Last used",
      width: "10%",
      render: (override) => (
        <LastUsedCell namespaceId={namespaceId} identifier={override.identifier} />
      ),
    },
    {
      key: "actions",
      header: "",
      width: "10%",
      render: (override) => (
        <OverridesTableAction
          overrideDetails={{
            duration: override.duration,
            limit: override.limit,
            async: override.async,
            overrideId: override.id,
          }}
          identifier={override.identifier}
          namespaceId={namespaceId}
        />
      ),
    },
  ];

  return (
    <>
      <VirtualTable
        data={overrides}
        isLoading={isLoading}
        columns={columns}
        keyExtractor={(override) => override.id}
        rowClassName={getRowClassName}
        onRowClick={handleRowClick}
        emptyState={
          <div className="w-full flex justify-center items-center h-full">
            <Empty className="w-[400px] flex items-start">
              <Empty.Icon className="w-auto" />
              <Empty.Title>No overrides found</Empty.Title>
              <Empty.Description className="text-left">
                No custom ratelimits found. Create your first override to get started.
              </Empty.Description>
            </Empty>
          </div>
        }
      />
      {selectedOverride && (
        <IdentifierDialog
          isModalOpen={isDialogOpen}
          onOpenChange={setIsDialogOpen}
          namespaceId={namespaceId}
          identifier={selectedOverride.identifier}
          overrideDetails={{
            overrideId: selectedOverride.id,
            limit: selectedOverride.limit,
            duration: selectedOverride.duration,
            async: selectedOverride.async,
          }}
        />
      )}
    </>
  );
};
