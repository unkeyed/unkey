"use client";
import { Badge } from "@/components/ui/badge";
import { VirtualTable } from "@/components/virtual-table";
import type { Column } from "@/components/virtual-table/types";
import { formatNumber } from "@/lib/fmt";
import { cn } from "@/lib/utils";
import { Empty } from "@unkey/ui";
import ms from "ms";
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
  ratelimits: Override[];
  lastUsedTimes: Record<string, number | null>;
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

export const OverridesTable = ({ namespaceId, ratelimits, lastUsedTimes }: Props) => {
  const columns: Column<Override>[] = [
    {
      key: "identifier",
      header: "Identifier",
      headerClassName: "pl-2",
      width: "25%",
      render: (override) => (
        <div className="flex flex-col items-start pl-2">
          <span className="font-mono text-xs text-gray-12 truncate max-w-40">
            {override.identifier}
          </span>
          <pre className="text-[11px]">{override.id}</pre>
        </div>
      ),
    },
    {
      key: "limits",
      header: "Limits",
      width: "25%",
      render: (override) => (
        <div className="inline-grid grid-cols-[1fr_auto_1fr] items-center gap-2">
          <div className="flex justify-start">
            <Badge
              className={cn(
                " px-2 rounded-md font-mono w-[100px] truncate",
                STATUS_STYLES.default.badge.default,
              )}
            >
              {formatNumber(override.limit)} Requests
            </Badge>
          </div>
          <span className="text-content-subtle">/</span>
          <div className="flex justify-start">
            <Badge
              className={cn(
                "uppercase px-2 rounded-md font-mono",
                STATUS_STYLES.default.badge.default,
              )}
            >
              {ms(override.duration)}
            </Badge>
          </div>
        </div>
      ),
    },
    {
      key: "async",
      header: "Async",
      width: "15%",
      render: (override) =>
        override.async === null ? (
          <div className="w-4 h-4 text-content-subtle">─</div>
        ) : (
          <Badge
            className={cn(
              "uppercase px-[6px] rounded-md font-mono min-w-[70px] inline-block text-center",
              STATUS_STYLES.default.badge.default,
            )}
          >
            {override.async ? "async" : "sync"}
          </Badge>
        ),
    },
    {
      key: "lastUsed",
      header: "Last used",
      width: "20%",
      render: (override) => {
        const lastUsed = lastUsedTimes[override.identifier];
        if (lastUsed) {
          return (
            <span className="font-mono text-xs text-content-subtle">
              {ms(Date.now() - lastUsed)} ago
            </span>
          );
        }
        return <div className="w-4 h-4 text-content-subtle">─</div>;
      },
    },
    {
      key: "actions",
      header: "",
      width: "15%",
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
    <VirtualTable
      data={ratelimits}
      columns={columns}
      keyExtractor={(override) => override.id}
      rowClassName={getRowClassName}
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
  );
};
