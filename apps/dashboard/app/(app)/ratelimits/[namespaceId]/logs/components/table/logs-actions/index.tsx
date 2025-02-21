import { toast } from "@/components/ui/toaster";
import { Clone, InputSearch } from "@unkey/icons";
import { type MenuItem, TableActionPopover } from "../../../../_components/table-action-popover";
import { useFilters } from "../../../hooks/use-filters";

export const LogsTableAction = ({ identifier }: { identifier: string }) => {
  const { filters, updateFilters } = useFilters();

  const items: MenuItem[] = [
    {
      id: "copy",
      label: "Copy identifier",
      icon: <Clone size="md-regular" />,
      onClick: (e) => {
        e.stopPropagation();
        navigator.clipboard.writeText(identifier);
        toast.success("Copied to clipboard", {
          description: identifier,
        });
      },
    },
    {
      id: "filter",
      label: "Filter for identifier",
      icon: <InputSearch />,
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

  return <TableActionPopover items={items} align="start" />;
};
