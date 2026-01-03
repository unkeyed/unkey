import { BarsFilter } from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";
import type { RatelimitOverviewFilterValue } from "../../../filters.schema";
import { useFilters } from "../../../hooks/use-filters";

type FilterPair = {
  status?: "blocked" | "passed";
  identifiers?: string;
};

export const InlineFilter = ({
  filterPair,
  content,
}: {
  filterPair: FilterPair;
  content: string;
}) => {
  const { filters, updateFilters } = useFilters();
  const fields = Object.entries(filterPair)
    .filter(([_, value]) => value !== undefined)
    .map(([key]) => key as keyof FilterPair);

  const activeFilters = filters.filter((f) =>
    ["startTime", "endTime", "since"].includes(f.field as keyof FilterPair),
  );

  return (
    <InfoTooltip
      variant="inverted"
      content={<span className="text-xs font-medium">{content}</span>}
    >
      <button
        onClick={() => {
          updateFilters([
            ...activeFilters,
            ...fields.map(
              (field) =>
                ({
                  field,
                  id: crypto.randomUUID(),
                  operator: "is",
                  value: filterPair[field] as string,
                }) satisfies RatelimitOverviewFilterValue,
            ),
          ]);
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
