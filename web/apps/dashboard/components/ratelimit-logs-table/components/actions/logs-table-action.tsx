"use client";

import { useFilters } from "@/app/(app)/[workspaceSlug]/ratelimits/[namespaceId]/logs/hooks/use-filters";
import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { toast } from "@unkey/ui";
import { IconCloneOutline18, IconInputSearchOutline18 } from "nucleo-ui-outline-18";

export const LogsTableAction = ({ identifier }: { identifier: string }) => {
  const { filters, updateFilters } = useFilters();

  const getLogsTableActionItems = (): MenuItem[] => {
    return [
      {
        id: "copy",
        label: "Copy identifier",
        icon: <IconCloneOutline18 className="size-4" />,
        onClick: (e) => {
          e.stopPropagation();
          navigator.clipboard
            .writeText(identifier)
            .then(() => {
              toast.success("Copied to clipboard", {
                description: identifier,
              });
            })
            .catch((error) => {
              console.error("Failed to copy to clipboard:", error);
              toast.error("Failed to copy to clipboard");
            });
        },
      },
      {
        id: "filter",
        label: "Filter for identifier",
        icon: <IconInputSearchOutline18 className="size-4" />,
        onClick: (e) => {
          e.stopPropagation();
          const newFilter = {
            id: crypto.randomUUID(),
            field: "identifiers" as const,
            operator: "is" as const,
            value: identifier,
          };
          const existingFilters = filters.filter(
            (f) => !(f.field === "identifiers" && f.value === identifier),
          );
          updateFilters([...existingFilters, newFilter]);
        },
      },
    ];
  };

  const menuItems = getLogsTableActionItems();

  return <TableActionPopover items={menuItems} align="start" />;
};
