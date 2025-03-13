import { useBookmarkedFilters } from "@/components/logs/hooks/use-bookmarked-filters";
import { QueriesPopover } from "@/components/logs/queries/queries-popover";
import { cn } from "@/lib/utils";
import { ChartBarAxisY } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useState } from "react";
import { useFilters } from "../../../../hooks/use-filters";

export const LogsQueries = () => {
  const { filters, updateFilters } = useFilters();
  const { savedFilters, toggleBookmark, applyFilterGroup } = useBookmarkedFilters({
    localStorageName: "auditSavedFilters",
    filters,
    updateFilters,
  });
  const [filterGroups, setfilterGroups] = useState(savedFilters);

  const handleApplyFilterGroup = (id: string) => {
    const group = savedFilters.find((filter) => filter.id === id);
    if (group) {
      applyFilterGroup(group);
    }
  };
  const handleGetFilters = () => {
    setfilterGroups(savedFilters);
  };
  return (
    <QueriesPopover
      savedFilters={filterGroups}
      toggleBookmark={toggleBookmark}
      applyFilterGroup={handleApplyFilterGroup}
      updateGroups={handleGetFilters}
    >
      <div className="group">
        <Button
          variant="ghost"
          size="md"
          className={cn("group-data-[state=open]:bg-gray-4 px-2 rounded-lg")}
          aria-label="Audit log queries"
          aria-haspopup="true"
          title="Press 'Q' to toggle queries"
        >
          <ChartBarAxisY size="md-regular" className="mt-1 ml-[3px] text-gray-9" />
          <span className="text-gray-12 font-medium text-[13px] leading-4">Queries</span>
        </Button>
      </div>
    </QueriesPopover>
  );
};
