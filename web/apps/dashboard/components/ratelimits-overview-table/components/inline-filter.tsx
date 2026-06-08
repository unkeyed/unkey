import type {
  RatelimitOverviewFilterField,
  RatelimitOverviewFilterValue,
} from "@/app/(app)/[workspaceSlug]/ratelimits/[namespaceId]/_overview/filters.schema";
import { useFilters } from "@/app/(app)/[workspaceSlug]/ratelimits/[namespaceId]/_overview/hooks/use-filters";
import { BarsFilter } from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";

type FilterPair = {
  status?: "blocked" | "passed";
  identifiers?: string;
};

// Time-range filters are preserved when applying an inline filter; everything
// else gets replaced by the new pair so the table doesn't accumulate stale rules.
const TIME_FIELDS: readonly RatelimitOverviewFilterField[] = ["startTime", "endTime", "since"];

export const InlineFilter = ({
  filterPair,
  content,
}: {
  filterPair: FilterPair;
  content: string;
}) => {
  const { filters, updateFilters } = useFilters();

  const activeFilters = filters.filter((f) => TIME_FIELDS.includes(f.field));

  return (
    <InfoTooltip
      asChild
      variant="inverted"
      content={<span className="text-xs font-medium">{content}</span>}
    >
      <button
        onClick={() => {
          const pairFilters: RatelimitOverviewFilterValue[] = [];
          if (filterPair.identifiers !== undefined) {
            pairFilters.push({
              field: "identifiers",
              id: crypto.randomUUID(),
              operator: "is",
              value: filterPair.identifiers,
            });
          }
          if (filterPair.status !== undefined) {
            pairFilters.push({
              field: "status",
              id: crypto.randomUUID(),
              operator: "is",
              value: filterPair.status,
            });
          }
          updateFilters([...activeFilters, ...pairFilters]);
        }}
        type="button"
      >
        <BarsFilter
          className="text-gray-12 invisible group-hover/identifier:visible"
          iconSize="md-medium"
        />
      </button>
    </InfoTooltip>
  );
};
