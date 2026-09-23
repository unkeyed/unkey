import { useFilters } from "@/app/(app)/[workspaceSlug]/audit/hooks/use-filters";
import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";
import { trpc } from "@/lib/trpc/client";

export const UsersFilter = () => {
  const { filters, updateFilters } = useFilters();
  const { data: users, isLoading } = trpc.audit.members.useQuery();

  if (isLoading) {
    return (
      <div className="flex flex-col gap-2 p-2">
        {Array.from({ length: 3 }).map((_, i) => (
          // biome-ignore lint/suspicious/noArrayIndexKey: safe to leave
          <div key={i} className="flex items-center gap-4.5 px-2 py-1">
            <div className="size-4 bg-grayA-3 rounded animate-pulse shrink-0" />
            <div className="h-4 w-[120px] bg-grayA-3 rounded animate-pulse" />
          </div>
        ))}
      </div>
    );
  }

  return (
    <FilterCheckbox
      showScroll
      options={(users ?? []).map((user, index) => ({
        ...user,
        checked: false,
        id: index,
      }))}
      filterField="users"
      checkPath="value"
      renderOptionContent={(checkbox) => (
        <div className="text-gray-12 text-xs">{checkbox.label}</div>
      )}
      createFilterValue={(option) => ({
        value: option.value,
      })}
      filters={filters}
      updateFilters={updateFilters}
    />
  );
};
