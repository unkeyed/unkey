import type {
  AuditLogsFilterField,
  AuditLogsFilterValue,
} from "@/app/(app)/[workspaceSlug]/audit/filters.schema";
import { useFilters } from "@/app/(app)/[workspaceSlug]/audit/hooks/use-filters";
import {
  type FilterSearchItem,
  createFilterOptionBuilder,
} from "@/components/logs/checkbox/filters-popover";
import { shortenId } from "@/lib/shorten-id";
import { trpc } from "@/lib/trpc/client";
import { useMemo } from "react";
import type { WorkspaceProps } from "../../../logs-client";
import { eventGroups } from "./components/events-filter";

const createAuditFilter = (field: AuditLogsFilterField, value: string): AuditLogsFilterValue => ({
  id: crypto.randomUUID(),
  field,
  operator: "is",
  value,
});

export function useAuditSearchItems(
  { rootKeys, buckets }: WorkspaceProps,
  isPopoverOpen: boolean,
): FilterSearchItem[] {
  const { filters, updateFilters } = useFilters();
  const { data: users } = trpc.audit.members.useQuery(undefined, { enabled: isPopoverOpen });

  return useMemo(() => {
    const option = createFilterOptionBuilder({
      filters,
      updateFilters,
      createFilter: createAuditFilter,
    });

    return [
      ...eventGroups.flatMap((group) =>
        group.items.map((item) =>
          option({
            field: "events",
            value: item.value,
            label: item.label,
            path: ["Events", group.label],
            keywords: [item.value],
          }),
        ),
      ),
      ...(users ?? []).map((user) =>
        option({ field: "users", value: user.value, label: user.label, path: ["Users"] }),
      ),
      ...rootKeys.map((rootKey) =>
        option({
          field: "rootKeys",
          value: rootKey.id,
          label: rootKey.name ?? shortenId(rootKey.id),
          path: ["Root Keys"],
          keywords: [rootKey.id],
        }),
      ),
      ...buckets.map((bucket) =>
        option({ field: "bucket", value: bucket, label: bucket, path: ["Bucket"], single: true }),
      ),
    ];
  }, [filters, updateFilters, users, rootKeys, buckets]);
}
