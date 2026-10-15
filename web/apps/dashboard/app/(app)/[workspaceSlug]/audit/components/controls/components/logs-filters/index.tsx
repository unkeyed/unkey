import type { AuditLogsFilterField } from "@/app/(app)/[workspaceSlug]/audit/filters.schema";
import { useFilters } from "@/app/(app)/[workspaceSlug]/audit/hooks/use-filters";
import { type FilterSearchItem, FiltersPopover } from "@/components/logs/checkbox/filters-popover";
import { shortenId } from "@/lib/shorten-id";
import { trpc } from "@/lib/trpc/client";
import { IconBarsFilterOutline18 } from "@unkey/icons";
import { Badge, Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useMemo, useState } from "react";
import type { WorkspaceProps } from "../../../logs-client";
import { BucketFilter } from "./components/bucket-filter";
import { EventsFilter, eventGroups } from "./components/events-filter";
import { RootKeysFilter } from "./components/root-keys-filter";
import { UsersFilter } from "./components/users-filter";

export const LogsFilters = (props: WorkspaceProps) => {
  const { filters, updateFilters } = useFilters();
  const [isPopoverOpen, setIsPopoverOpen] = useState(false);
  const { data: users } = trpc.audit.members.useQuery(undefined, { enabled: isPopoverOpen });

  const searchItems = useMemo<FilterSearchItem[]>(() => {
    const isSelected = (field: AuditLogsFilterField, value: string) =>
      filters.some((filter) => filter.field === field && filter.value === value);

    const toggle = (field: AuditLogsFilterField, value: string) => {
      const selected = isSelected(field, value);
      const otherFilters = filters.filter(
        (filter) => filter.field !== field || (field !== "bucket" && filter.value !== value),
      );
      updateFilters(
        selected
          ? otherFilters
          : [...otherFilters, { id: crypto.randomUUID(), field, operator: "is", value }],
      );
    };

    const option = (
      field: AuditLogsFilterField,
      value: string,
      item: Pick<FilterSearchItem, "label" | "path" | "keywords" | "description">,
    ): FilterSearchItem => ({
      kind: "option",
      id: `${field}:${value}`,
      checked: isSelected(field, value),
      onSelect: () => toggle(field, value),
      ...item,
    });

    return [
      ...eventGroups.flatMap((group) =>
        group.items.map((item) =>
          option("events", item.value, {
            label: item.label,
            path: ["Events", group.label],
            keywords: [item.value],
          }),
        ),
      ),
      ...(users ?? []).map((user) =>
        option("users", user.value, { label: user.label, path: ["Users"] }),
      ),
      ...props.rootKeys.map((rootKey) =>
        option("rootKeys", rootKey.id, {
          label: rootKey.name ?? shortenId(rootKey.id),
          path: ["Root Keys"],
          keywords: [rootKey.id],
        }),
      ),
      ...props.buckets.map((bucket) =>
        option("bucket", bucket, { label: bucket, path: ["Bucket"] }),
      ),
    ];
  }, [filters, updateFilters, users, props.rootKeys, props.buckets]);

  return (
    <FiltersPopover
      items={[
        {
          id: "events",
          label: "Events",
          shortcut: "e",
          component: <EventsFilter />,
        },
        {
          id: "users",
          label: "Users",
          shortcut: "m",
          component: <UsersFilter />,
        },
        {
          id: "rootKeys",
          label: "Root Keys",
          shortcut: "p",
          component: <RootKeysFilter rootKeys={props.rootKeys} />,
        },
        {
          id: "bucket",
          label: "Bucket",
          shortcut: "b",
          component: <BucketFilter buckets={props.buckets} />,
        },
      ]}
      searchItems={searchItems}
      searchPlaceholder="Search events, users, root keys..."
      activeFilters={filters}
      open={isPopoverOpen}
      onOpenChange={setIsPopoverOpen}
    >
      <div className="group">
        <Button
          variant="ghost"
          size="md"
          className={cn(
            "group-data-popup-open:bg-gray-4 px-2 rounded-lg",
            filters.length > 0 ? "bg-gray-4" : "",
          )}
          aria-label="Filter logs"
          aria-haspopup="true"
          title="Press 'F' to toggle filters"
        >
          <IconBarsFilterOutline18 className="text-gray-9 size-4" />
          <span className="text-gray-12 font-medium text-sm">Filter</span>
          {filters.length > 0 && <Badge variant="count">{filters.length}</Badge>}
        </Button>
      </div>
    </FiltersPopover>
  );
};
