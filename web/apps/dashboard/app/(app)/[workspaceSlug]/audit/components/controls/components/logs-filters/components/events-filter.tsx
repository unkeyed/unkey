import { useFilters } from "@/app/(app)/[workspaceSlug]/audit/hooks/use-filters";
import {
  type CheckboxGroup,
  GroupedFilterCheckbox,
} from "@/components/logs/checkbox/grouped-filter-checkbox";
import { unkeyAuditLogEvents } from "@unkey/schema/src/auditlog";

export const eventGroups: CheckboxGroup[] = [
  ...unkeyAuditLogEvents.options
    .reduce((groups, event) => {
      const [resource] = event.split(".");
      const items = groups.get(resource) ?? [];
      items.push({ value: event, label: event.slice(resource.length + 1) });
      return groups.set(resource, items);
    }, new Map<string, CheckboxGroup["items"]>())
    .entries(),
].map(([resource, items]) => ({ id: resource, label: resource, items }));

export const EventsFilter = () => {
  const { filters, updateFilters } = useFilters();
  return (
    <GroupedFilterCheckbox
      groups={eventGroups}
      filterField="events"
      filters={filters}
      updateFilters={updateFilters}
      createFilter={(value) => ({
        id: crypto.randomUUID(),
        field: "events" as const,
        operator: "is" as const,
        value,
      })}
      searchPlaceholder="Search events..."
    />
  );
};
