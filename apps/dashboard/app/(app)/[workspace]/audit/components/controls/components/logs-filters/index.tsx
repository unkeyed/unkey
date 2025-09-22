import { useFilters } from "@/app/(app)/[workspace]/audit/hooks/use-filters";
import { FiltersPopover } from "@/components/logs/checkbox/filters-popover";
import { BarsFilter } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import type { WorkspaceProps } from "../../../logs-client";
import { BucketFilter } from "./components/bucket-filter";
import { EventsFilter } from "./components/events-filter";
import { RootKeysFilter } from "./components/root-keys-filter";
import { UsersFilter } from "./components/users-filter";

export const LogsFilters = (props: WorkspaceProps) => {
  const { filters } = useFilters();
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
          component: <UsersFilter users={props.members} />,
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
      activeFilters={filters}
    >
      <div className="group">
        <Button
          variant="ghost"
          className={cn(
            "group-data-[state=open]:bg-gray-4 px-2",
            filters.length > 0 ? "bg-gray-4" : "",
          )}
          aria-label="Filter logs"
          aria-haspopup="true"
          title="Press 'F' to toggle filters"
        >
          <BarsFilter className="text-accent-9 size-4" />
          <span className="text-accent-12 font-medium text-[13px]">Filter</span>
          {filters.length > 0 && (
            <div className="bg-gray-7 rounded h-4 px-1 text-[11px] font-medium text-accent-12 text-center flex items-center justify-center">
              {filters.length}
            </div>
          )}
        </Button>
      </div>
    </FiltersPopover>
  );
};
