import { useFilters } from "@/app/(app)/[workspace]/audit/hooks/use-filters";
import { FilterCheckbox } from "@/components/logs/checkbox/filter-checkbox";
import { unkeyAuditLogEvents } from "@unkey/schema/src/auditlog";

type EventsOption = {
  id: number;
  display: string;
  label: string;
  checked: boolean;
};
export const EventsFilter = () => {
  const { filters, updateFilters } = useFilters();
  return (
    <FilterCheckbox
      showScroll
      options={Object.values(unkeyAuditLogEvents.Values).map<EventsOption>((value, index) => ({
        id: index,
        display: value,
        label: value,
        checked: false,
      }))}
      filterField="events"
      checkPath="display"
      renderOptionContent={(checkbox) => (
        <div className="text-accent-12 text-xs">{checkbox.display}</div>
      )}
      createFilterValue={(option) => ({
        value: option.display,
      })}
      filters={filters}
      updateFilters={updateFilters}
    />
  );
};
