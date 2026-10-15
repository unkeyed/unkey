import { useFilters } from "@/app/(app)/[workspaceSlug]/audit/hooks/use-filters";
import { FiltersPopover } from "@/components/logs/checkbox/filters-popover";
import { IconBarsFilterOutline18 } from "@unkey/icons";
import { Badge, Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useState } from "react";
import type { WorkspaceProps } from "../../../logs-client";
import { BucketFilter } from "./components/bucket-filter";
import { EventsFilter } from "./components/events-filter";
import { RootKeysFilter } from "./components/root-keys-filter";
import { UsersFilter } from "./components/users-filter";
import { useAuditSearchItems } from "./use-audit-search-items";

export const LogsFilters = (props: WorkspaceProps) => {
  const { filters } = useFilters();
  const [isPopoverOpen, setIsPopoverOpen] = useState(false);

  const searchItems = useAuditSearchItems(props, isPopoverOpen);

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
