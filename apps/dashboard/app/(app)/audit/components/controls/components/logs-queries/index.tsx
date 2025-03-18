import { useBookmarkedFilters } from "@/components/logs/hooks/use-bookmarked-filters";
import type { ParsedSavedFiltersType } from "@/components/logs/hooks/use-bookmarked-filters";
import { QueriesPopover } from "@/components/logs/queries/queries-popover";
import { cn } from "@/lib/utils";
import { ChartBarAxisY } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useEffect, useState } from "react";
import { useFilters } from "../../../../hooks/use-filters";

export const LogsQueries = () => {
  const { filters, updateFilters } = useFilters();
  const { parseSavedFilters, savedFilters, toggleBookmark, applyFilterGroup } =
    useBookmarkedFilters({
      localStorageName: "auditSavedFilters",
      filters,
      updateFilters,
    });
  const [filterGroups, setfilterGroups] = useState<ParsedSavedFiltersType[]>(parseSavedFilters);

  function handleBookmarkTooggle(groupId: string) {
    toggleBookmark(groupId);
    const newFilters: ParsedSavedFiltersType[] = parseSavedFilters();
    setfilterGroups(newFilters);
  }

  useEffect(() => {
    const newFilters = parseSavedFilters();
    if (JSON.stringify(newFilters) === JSON.stringify(filterGroups)) {
      return;
    }
    setfilterGroups(newFilters);
  });

  const handleApplyFilterGroup = (groupId: string) => {
    const group = savedFilters.find((group) => group.id === groupId);
    if (!group) {
      return;
    }
    applyFilterGroup(group);
  };

  return (
    <QueriesPopover
      filterGroups={filterGroups || []}
      toggleBookmark={handleBookmarkTooggle}
      applyFilterGroup={handleApplyFilterGroup}
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
