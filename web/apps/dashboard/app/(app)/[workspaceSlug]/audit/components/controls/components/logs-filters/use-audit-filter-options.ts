import type { FilterItemOption } from "@/components/logs/checkbox/filters-popover";
import { shortenId } from "@/lib/shorten-id";
import { trpc } from "@/lib/trpc/client";
import { useMemo } from "react";
import type { WorkspaceProps } from "../../../logs-client";
import { eventGroups } from "./components/events-filter";

const eventOptions: FilterItemOption<string>[] = eventGroups.flatMap((group) =>
  group.items.map((item) => ({
    value: item.value,
    label: item.label,
    path: ["Events", group.label],
    keywords: [item.value],
  })),
);

export function useAuditFilterOptions(
  { rootKeys, buckets }: WorkspaceProps,
  isPopoverOpen: boolean,
) {
  const { data: users } = trpc.audit.members.useQuery(undefined, { enabled: isPopoverOpen });

  return useMemo(
    () => ({
      events: eventOptions,
      users: (users ?? []).map((user) => ({ value: user.value, label: user.label })),
      rootKeys: rootKeys.map((rootKey) => ({
        value: rootKey.id,
        label: rootKey.name ?? shortenId(rootKey.id),
        keywords: [rootKey.id],
      })),
      buckets: buckets.map((bucket) => ({ value: bucket, label: bucket })),
    }),
    [users, rootKeys, buckets],
  );
}
