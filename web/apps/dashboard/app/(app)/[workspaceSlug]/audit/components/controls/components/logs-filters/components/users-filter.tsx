import { useFilters } from "@/app/(app)/[workspaceSlug]/audit/hooks/use-filters";
import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";
import { trpc } from "@/lib/trpc/client";
import { getErrorMessage } from "@/lib/unkey-client";
import { Button } from "@unkey/ui";

export const UsersFilter = () => {
  const { filters, updateFilters } = useFilters();
  const { data: users, isLoading, isError, error, refetch } = trpc.audit.members.useQuery();

  const retry = () => {
    refetch().catch((retryError: unknown) => {
      console.error("Failed to retry audit members query", retryError);
    });
  };

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

  if (isError) {
    return (
      <div className="flex flex-col items-center gap-3 px-4 py-6 text-center">
        <span role="alert" className="text-gray-11 text-xs">
          {getErrorMessage(error, "We couldn't load workspace members.")}
        </span>
        <Button variant="outline" onClick={retry}>
          Retry
        </Button>
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
