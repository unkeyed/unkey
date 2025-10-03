"use client";

import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { Clone, InputSearch } from "@unkey/icons";
import { toast } from "@unkey/ui";
import { useFilters } from "../../../hooks/use-filters";

export const LogsTableAction = ({ identifier }: { identifier: string }) => {
  const { filters, updateFilters } = useFilters();

  const getLogsTableActionItems = (): MenuItem[] => {
    return [
      {
        id: "copy",
        label: "Copy identifier",
        icon: <Clone iconsize="md-medium" />,
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
        icon: <InputSearch iconsize="md-medium" />,
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
